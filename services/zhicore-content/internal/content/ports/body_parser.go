package ports

import (
	"context"
	"fmt"
)

// BodyParserRegistry is consumed by application use cases so body schema
// selection stays outside handlers and parser implementations.
type BodyParserRegistry interface {
	Parse(ctx context.Context, input PostBodyWriteInput) (NormalizedBody, error)
}

type PostBodyWriteInput struct {
	SchemaVersion int    `json:"schemaVersion"`
	Blocks        Blocks `json:"blocks"`
}

// BodyValidationPolicy is injected from runtime configuration so request
// parsing limits are explicit business safeguards, not hidden parser globals.
type BodyValidationPolicy struct {
	MaxCanonicalJSONBytes int
	MaxPlainTextChars     int
	MaxBlocks             int
	MaxInlineNodes        int
	MaxContainerDepth     int
	MaxTableCells         int
	MaxLatexChars         int
	MaxCodeBlockChars     int
	MaxExternalLinks      int
	MaxValidationErrors   int
}

type NormalizedBody struct {
	PlainText     string
	MediaRefs     []MediaRef
	ExternalLinks []string
	CanonicalJSON []byte
	ContentHash   string
	SizeBytes     int
	BlockCount    int
}

type MediaRef struct {
	FileID string
}

type ValidationDetail struct {
	Path string
	Code string
}

type BodyValidationError struct {
	Details   []ValidationDetail
	Truncated bool
}

func (e *BodyValidationError) Error() string {
	if len(e.Details) == 0 {
		return "body validation failed"
	}
	return fmt.Sprintf("body validation failed: %s at %s", e.Details[0].Code, e.Details[0].Path)
}
