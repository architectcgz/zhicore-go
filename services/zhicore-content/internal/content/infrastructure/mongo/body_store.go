package mongo

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/domain"
	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
	"go.mongodb.org/mongo-driver/v2/bson"
	drivermongo "go.mongodb.org/mongo-driver/v2/mongo"
)

var errNoDocuments = drivermongo.ErrNoDocuments

type BodyIDGenerator interface {
	NewID() (string, error)
}

type bodyCollection interface {
	InsertOne(ctx context.Context, document bodyDocument) error
	FindOne(ctx context.Context, bodyID string) (bodyDocument, error)
	DeleteOne(ctx context.Context, bodyID string) (int64, error)
}

type BodyStore struct {
	collection bodyCollection
	ids        BodyIDGenerator
}

func NewBodyStore(collection *drivermongo.Collection, ids BodyIDGenerator) *BodyStore {
	return newBodyStore(driverBodyCollection{collection: collection}, ids)
}

func newBodyStore(collection bodyCollection, ids BodyIDGenerator) *BodyStore {
	if ids == nil {
		ids = randomBodyIDGenerator{}
	}
	return &BodyStore{collection: collection, ids: ids}
}

func (s *BodyStore) WriteDraftBody(ctx context.Context, input ports.WriteBodyInput) (ports.StoredBody, error) {
	return s.writeBody(ctx, "DRAFT", input)
}

func (s *BodyStore) WriteSnapshotBody(ctx context.Context, input ports.WriteBodyInput) (ports.StoredBody, error) {
	return s.writeBody(ctx, "SNAPSHOT", input)
}

func (s *BodyStore) ReadBody(ctx context.Context, bodyID string) (ports.StoredBody, error) {
	if err := ctx.Err(); err != nil {
		return ports.StoredBody{}, err
	}

	document, err := s.collection.FindOne(ctx, bodyID)
	if errors.Is(err, errNoDocuments) {
		return ports.StoredBody{}, domain.ErrBodyUnavailable
	}
	if err != nil {
		return ports.StoredBody{}, fmt.Errorf("read content body %s: %w", bodyID, err)
	}
	if document.ID == "" {
		document.ID = bodyID
	}
	if err := verifyDocumentHash(document); err != nil {
		return ports.StoredBody{}, err
	}

	blocks, err := decodeDocumentBlocks(document)
	if err != nil {
		return ports.StoredBody{}, err
	}
	return ports.StoredBody{
		ID:            document.ID,
		SchemaVersion: document.SchemaVersion,
		Blocks:        blocks,
		CanonicalJSON: append([]byte(nil), document.CanonicalJSON...),
		PlainText:     document.PlainText,
		ContentHash:   document.ContentHash,
		SizeBytes:     document.SizeBytes,
		BlockCount:    document.BlockCount,
		CreatedAt:     document.CreatedAt,
	}, nil
}

func (s *BodyStore) DeleteBody(ctx context.Context, bodyID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if _, err := s.collection.DeleteOne(ctx, bodyID); err != nil {
		return fmt.Errorf("delete content body %s: %w", bodyID, err)
	}
	return nil
}

func (s *BodyStore) writeBody(ctx context.Context, role string, input ports.WriteBodyInput) (ports.StoredBody, error) {
	if err := ctx.Err(); err != nil {
		return ports.StoredBody{}, err
	}
	if expected := hashCanonicalJSON(input.CanonicalJSON); input.ContentHash != "" && input.ContentHash != expected {
		return ports.StoredBody{}, domain.ErrBodyInconsistent
	}
	blocks, err := blocksDocumentFromCanonical(input.CanonicalJSON)
	if err != nil {
		return ports.StoredBody{}, err
	}

	bodyID, err := s.ids.NewID()
	if err != nil {
		return ports.StoredBody{}, fmt.Errorf("generate content body id: %w", err)
	}
	document := bodyDocument{
		ID:            bodyID,
		PostPublicID:  input.PostPublicID,
		OwnerID:       input.OwnerID,
		BodyRole:      role,
		SchemaVersion: input.SchemaVersion,
		Format:        "blocks",
		Blocks:        blocks,
		CanonicalJSON: append([]byte(nil), input.CanonicalJSON...),
		PlainText:     input.PlainText,
		ContentHash:   input.ContentHash,
		SizeBytes:     input.SizeBytes,
		BlockCount:    input.BlockCount,
		CreatedAt:     input.CreatedAt,
	}
	if document.ContentHash == "" {
		document.ContentHash = hashCanonicalJSON(document.CanonicalJSON)
	}
	if err := s.collection.InsertOne(ctx, document); err != nil {
		return ports.StoredBody{}, fmt.Errorf("insert content %s body: %w", role, err)
	}
	return ports.StoredBody{
		ID:            document.ID,
		SchemaVersion: document.SchemaVersion,
		Blocks:        cloneBlocks(input.Blocks),
		CanonicalJSON: append([]byte(nil), document.CanonicalJSON...),
		PlainText:     document.PlainText,
		ContentHash:   document.ContentHash,
		SizeBytes:     document.SizeBytes,
		BlockCount:    document.BlockCount,
		CreatedAt:     document.CreatedAt,
	}, nil
}

type bodyDocument struct {
	ID            string    `bson:"_id"`
	PostPublicID  string    `bson:"postPublicId,omitempty"`
	OwnerID       int64     `bson:"ownerId,omitempty"`
	BodyRole      string    `bson:"bodyRole,omitempty"`
	SchemaVersion int       `bson:"schemaVersion"`
	Format        string    `bson:"format"`
	Blocks        []any     `bson:"blocks"`
	CanonicalJSON []byte    `bson:"canonicalJson"`
	PlainText     string    `bson:"plainText"`
	ContentHash   string    `bson:"contentHash"`
	SizeBytes     int       `bson:"sizeBytes"`
	BlockCount    int       `bson:"blockCount"`
	CreatedAt     time.Time `bson:"createdAt"`
}

type canonicalBody struct {
	SchemaVersion int          `json:"schemaVersion"`
	Blocks        ports.Blocks `json:"blocks"`
}

type bodyDocumentShape struct {
	SchemaVersion int   `json:"schemaVersion"`
	Blocks        []any `json:"blocks"`
}

func verifyDocumentHash(document bodyDocument) error {
	if document.ContentHash != hashCanonicalJSON(document.CanonicalJSON) {
		return domain.ErrBodyInconsistent
	}
	if document.Format != "" && document.Format != "blocks" {
		return domain.ErrBodyInconsistent
	}
	return nil
}

func blocksDocumentFromCanonical(canonicalJSON []byte) ([]any, error) {
	var body bodyDocumentShape
	if err := json.Unmarshal(canonicalJSON, &body); err != nil {
		return nil, fmt.Errorf("%w: decode canonical body", domain.ErrBodyInconsistent)
	}
	if body.Blocks == nil {
		return nil, domain.ErrBodyInconsistent
	}
	return body.Blocks, nil
}

func decodeDocumentBlocks(document bodyDocument) (ports.Blocks, error) {
	rawBody, err := json.Marshal(bodyDocumentShape{
		SchemaVersion: document.SchemaVersion,
		Blocks:        document.Blocks,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: encode document body", domain.ErrBodyInconsistent)
	}

	var canonical canonicalBody
	if err := json.Unmarshal(rawBody, &canonical); err != nil {
		return nil, fmt.Errorf("%w: decode document blocks", domain.ErrBodyInconsistent)
	}
	return canonical.Blocks, nil
}

func hashCanonicalJSON(canonical []byte) string {
	sum := sha256.Sum256(canonical)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func cloneBlocks(blocks ports.Blocks) ports.Blocks {
	if blocks == nil {
		return nil
	}
	cloned := make(ports.Blocks, len(blocks))
	copy(cloned, blocks)
	return cloned
}

type driverBodyCollection struct {
	collection *drivermongo.Collection
}

func (c driverBodyCollection) InsertOne(ctx context.Context, document bodyDocument) error {
	_, err := c.collection.InsertOne(ctx, document)
	return err
}

func (c driverBodyCollection) FindOne(ctx context.Context, bodyID string) (bodyDocument, error) {
	var document bodyDocument
	err := c.collection.FindOne(ctx, bson.M{"_id": bodyID}).Decode(&document)
	return document, err
}

func (c driverBodyCollection) DeleteOne(ctx context.Context, bodyID string) (int64, error) {
	result, err := c.collection.DeleteOne(ctx, bson.M{"_id": bodyID})
	if err != nil {
		return 0, err
	}
	return result.DeletedCount, nil
}

type randomBodyIDGenerator struct{}

func (randomBodyIDGenerator) NewID() (string, error) {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}
	return "body_" + hex.EncodeToString(buf[:]), nil
}

var _ ports.PostContentStore = (*BodyStore)(nil)
