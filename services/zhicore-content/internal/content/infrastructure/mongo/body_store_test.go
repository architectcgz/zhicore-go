package mongo

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"
	"time"

	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/domain"
	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
)

func TestBodyStoreWriteDraftAndSnapshotBodies(t *testing.T) {
	now := time.Date(2026, 7, 5, 13, 0, 0, 0, time.UTC)
	canonical := []byte(`{"schemaVersion":1,"blocks":[]}`)
	collection := &fakeBodyCollection{}
	store := newBodyStore(collection, newFixedBodyIDGenerator("body_draft", "body_snapshot"))

	draft, err := store.WriteDraftBody(context.Background(), ports.WriteBodyInput{
		PostPublicID:  "post_pub_1",
		OwnerID:       42,
		SchemaVersion: 1,
		Blocks:        ports.Blocks{},
		CanonicalJSON: canonical,
		PlainText:     "hello",
		ContentHash:   contentHash(canonical),
		SizeBytes:     len(canonical),
		BlockCount:    0,
		CreatedAt:     now,
	})
	if err != nil {
		t.Fatalf("WriteDraftBody() error = %v", err)
	}
	snapshot, err := store.WriteSnapshotBody(context.Background(), ports.WriteBodyInput{
		PostPublicID:  "post_pub_1",
		OwnerID:       42,
		SchemaVersion: 1,
		Blocks:        ports.Blocks{},
		CanonicalJSON: canonical,
		PlainText:     "hello",
		ContentHash:   contentHash(canonical),
		SizeBytes:     len(canonical),
		BlockCount:    0,
		CreatedAt:     now,
	})
	if err != nil {
		t.Fatalf("WriteSnapshotBody() error = %v", err)
	}
	if draft.ID != "body_draft" || snapshot.ID != "body_snapshot" {
		t.Fatalf("body ids = %q/%q, want generated ids", draft.ID, snapshot.ID)
	}
	if len(collection.inserted) != 2 {
		t.Fatalf("inserted documents = %d, want 2", len(collection.inserted))
	}
	if collection.inserted[0].BodyRole != "DRAFT" || collection.inserted[1].BodyRole != "SNAPSHOT" {
		t.Fatalf("roles = %q/%q, want DRAFT/SNAPSHOT", collection.inserted[0].BodyRole, collection.inserted[1].BodyRole)
	}
	if collection.inserted[0].Format != "blocks" || collection.inserted[1].Format != "blocks" {
		t.Fatalf("formats = %q/%q, want blocks", collection.inserted[0].Format, collection.inserted[1].Format)
	}
	if len(collection.inserted[0].Blocks) != 0 || collection.inserted[0].CanonicalJSON == nil {
		t.Fatalf("stored body shape = %+v, want top-level blocks plus canonical json", collection.inserted[0])
	}
	if collection.inserted[0].PostPublicID != "post_pub_1" || collection.inserted[0].OwnerID != 42 {
		t.Fatalf("document owner fields = %+v", collection.inserted[0])
	}
}

func TestBodyStoreReadBodyReturnsBlocksAndVerifiesHash(t *testing.T) {
	canonical := []byte(`{"schemaVersion":1,"blocks":[{"type":"paragraph","children":[{"type":"text","text":"hello"}]}]}`)
	createdAt := time.Date(2026, 7, 5, 13, 30, 0, 0, time.UTC)
	collection := &fakeBodyCollection{
		findDocument: &bodyDocument{
			ID:            "body_1",
			SchemaVersion: 1,
			Format:        "blocks",
			Blocks:        mustBlocksDocument(t, canonical),
			CanonicalJSON: canonical,
			PlainText:     "hello",
			ContentHash:   contentHash(canonical),
			SizeBytes:     len(canonical),
			BlockCount:    1,
			CreatedAt:     createdAt,
		},
	}
	store := newBodyStore(collection, newFixedBodyIDGenerator())

	body, err := store.ReadBody(context.Background(), "body_1")
	if err != nil {
		t.Fatalf("ReadBody() error = %v", err)
	}
	if body.ID != "body_1" || body.ContentHash != contentHash(canonical) || body.CreatedAt != createdAt {
		t.Fatalf("body = %+v, want decoded stored body", body)
	}
	if len(body.Blocks) != 1 || body.Blocks[0].Kind() != ports.BlockParagraph {
		t.Fatalf("blocks = %#v, want decoded paragraph", body.Blocks)
	}
	if collection.lastFilterID != "body_1" {
		t.Fatalf("find filter id = %q, want body_1", collection.lastFilterID)
	}
}

func TestBodyStoreReadBodyMapsMissingAndHashMismatch(t *testing.T) {
	t.Run("missing body", func(t *testing.T) {
		collection := &fakeBodyCollection{findErr: errNoDocuments}
		store := newBodyStore(collection, newFixedBodyIDGenerator())

		_, err := store.ReadBody(context.Background(), "missing")
		if !errors.Is(err, domain.ErrBodyUnavailable) {
			t.Fatalf("ReadBody() error = %v, want ErrBodyUnavailable", err)
		}
	})

	t.Run("hash mismatch", func(t *testing.T) {
		canonical := []byte(`{"schemaVersion":1,"blocks":[]}`)
		collection := &fakeBodyCollection{
			findDocument: &bodyDocument{
				ID:            "body_bad",
				SchemaVersion: 1,
				Format:        "blocks",
				Blocks:        mustBlocksDocument(t, canonical),
				CanonicalJSON: canonical,
				ContentHash:   "sha256:wrong",
			},
		}
		store := newBodyStore(collection, newFixedBodyIDGenerator())

		_, err := store.ReadBody(context.Background(), "body_bad")
		if !errors.Is(err, domain.ErrBodyInconsistent) {
			t.Fatalf("ReadBody() error = %v, want ErrBodyInconsistent", err)
		}
	})
}

func TestBodyStoreHonorsContextCancellationBeforeWrite(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	collection := &fakeBodyCollection{}
	store := newBodyStore(collection, newFixedBodyIDGenerator("body_cancel"))

	_, err := store.WriteDraftBody(ctx, ports.WriteBodyInput{SchemaVersion: 1})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("WriteDraftBody() error = %v, want context.Canceled", err)
	}
	if len(collection.inserted) != 0 {
		t.Fatalf("inserted documents = %d, want none", len(collection.inserted))
	}
}

func TestBodyStoreDeleteBodyIsPreciseAndIdempotent(t *testing.T) {
	collection := &fakeBodyCollection{deleteCount: 0}
	store := newBodyStore(collection, newFixedBodyIDGenerator())

	if err := store.DeleteBody(context.Background(), "body_1"); err != nil {
		t.Fatalf("DeleteBody() error = %v", err)
	}
	if collection.lastFilterID != "body_1" {
		t.Fatalf("delete filter id = %q, want body_1", collection.lastFilterID)
	}
}

type fakeBodyCollection struct {
	inserted     []bodyDocument
	findDocument *bodyDocument
	findErr      error
	deleteCount  int64
	lastFilterID string
}

func (c *fakeBodyCollection) InsertOne(ctx context.Context, document bodyDocument) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	c.inserted = append(c.inserted, document)
	return nil
}

func (c *fakeBodyCollection) FindOne(ctx context.Context, bodyID string) (bodyDocument, error) {
	if err := ctx.Err(); err != nil {
		return bodyDocument{}, err
	}
	c.lastFilterID = bodyID
	if c.findErr != nil {
		return bodyDocument{}, c.findErr
	}
	return *c.findDocument, nil
}

func (c *fakeBodyCollection) DeleteOne(ctx context.Context, bodyID string) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	c.lastFilterID = bodyID
	return c.deleteCount, nil
}

type fixedBodyIDGenerator []string

func newFixedBodyIDGenerator(ids ...string) *fixedBodyIDGenerator {
	generator := fixedBodyIDGenerator(ids)
	return &generator
}

func (g *fixedBodyIDGenerator) NewID() (string, error) {
	if len(*g) == 0 {
		return "", errors.New("body id sequence exhausted")
	}
	id := (*g)[0]
	*g = (*g)[1:]
	return id, nil
}

func contentHash(canonical []byte) string {
	sum := sha256.Sum256(canonical)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func mustBlocksDocument(t *testing.T, canonical []byte) []any {
	t.Helper()
	blocks, err := blocksDocumentFromCanonical(canonical)
	if err != nil {
		t.Fatalf("blocksDocumentFromCanonical() error = %v", err)
	}
	return blocks
}
