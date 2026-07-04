package body

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
)

type BodyValidationPolicy = ports.BodyValidationPolicy
type PostBodyWriteInput = ports.PostBodyWriteInput
type Blocks = ports.Blocks
type ParagraphBlock = ports.ParagraphBlock
type HeadingBlock = ports.HeadingBlock
type CodeBlock = ports.CodeBlock
type TableBlock = ports.TableBlock
type ImageBlock = ports.ImageBlock
type ExternalEmbedBlock = ports.ExternalEmbedBlock
type UnsupportedBlock = ports.UnsupportedBlock
type BlockType = ports.BlockType
type InlineNode = ports.InlineNode
type InlineMark = ports.InlineMark
type TableCell = ports.TableCell
type BodyValidationError = ports.BodyValidationError

func testPolicy() BodyValidationPolicy {
	return BodyValidationPolicy{
		MaxCanonicalJSONBytes: 1024,
		MaxPlainTextChars:     200,
		MaxBlocks:             8,
		MaxInlineNodes:        16,
		MaxContainerDepth:     2,
		MaxTableCells:         16,
		MaxLatexChars:         32,
		MaxCodeBlockChars:     64,
		MaxExternalLinks:      4,
		MaxValidationErrors:   3,
	}
}

func TestV1BodyParserNormalizesSafeBlocks(t *testing.T) {
	parser := NewV1BodyParser(testPolicy())

	normalized, err := parser.Parse(context.Background(), PostBodyWriteInput{
		SchemaVersion: 1,
		Blocks: Blocks{
			&ParagraphBlock{
				Children: []InlineNode{
					{
						Type: "text",
						Text: "安全链接",
						Marks: []InlineMark{
							{Type: "link", Href: "https://example.com/path?q=1"},
						},
					},
				},
			},
			&ImageBlock{
				FileID: "file_123",
				Alt:    "示例图",
			},
			&ExternalEmbedBlock{
				Provider: "image",
				URL:      "https://cdn.example.com/image.png",
				Title:    "外部图",
			},
		},
	})

	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if normalized.PlainText != "安全链接" {
		t.Fatalf("PlainText = %q, want %q", normalized.PlainText, "安全链接")
	}
	if normalized.ContentHash == "" || !strings.HasPrefix(normalized.ContentHash, "sha256:") {
		t.Fatalf("ContentHash = %q, want sha256 hash", normalized.ContentHash)
	}
	if normalized.SizeBytes != len(normalized.CanonicalJSON) {
		t.Fatalf("SizeBytes = %d, want canonical JSON length %d", normalized.SizeBytes, len(normalized.CanonicalJSON))
	}
	if len(normalized.MediaRefs) != 1 || normalized.MediaRefs[0].FileID != "file_123" {
		t.Fatalf("MediaRefs = %#v, want file_123", normalized.MediaRefs)
	}
	if len(normalized.ExternalLinks) != 2 {
		t.Fatalf("ExternalLinks = %#v, want 2 links", normalized.ExternalLinks)
	}
}

func TestV1BodyParserNormalizesDecodedJSONBody(t *testing.T) {
	parser := NewV1BodyParser(testPolicy())
	var input PostBodyWriteInput
	err := json.Unmarshal([]byte(`{
		"schemaVersion": 1,
		"blocks": [
			{
				"type": "paragraph",
				"children": [
					{
						"type": "text",
						"text": "安全链接",
						"marks": [
							{"type": "link", "href": "https://example.com/from-json"}
						]
					}
				]
			}
		]
	}`), &input)
	if err != nil {
		t.Fatalf("json.Unmarshal returned error: %v", err)
	}

	normalized, err := parser.Parse(context.Background(), input)

	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if normalized.PlainText != "安全链接" {
		t.Fatalf("PlainText = %q, want decoded JSON text", normalized.PlainText)
	}
	if len(normalized.ExternalLinks) != 1 || normalized.ExternalLinks[0] != "https://example.com/from-json" {
		t.Fatalf("ExternalLinks = %#v, want decoded JSON link", normalized.ExternalLinks)
	}
}

func TestBlocksMarshalJSONInjectsBlockType(t *testing.T) {
	raw, err := json.Marshal(Blocks{
		&ParagraphBlock{
			Children: []InlineNode{{Type: "text", Text: "正文"}},
		},
	})
	if err != nil {
		t.Fatalf("json.Marshal returned error: %v", err)
	}

	body := string(raw)
	if !strings.Contains(body, `"type":"paragraph"`) {
		t.Fatalf("Blocks JSON = %s, want paragraph type", body)
	}
	if strings.Contains(body, `"Block"`) {
		t.Fatalf("Blocks JSON = %s, want flattened block fields", body)
	}
}

func TestV1BodyParserCanonicalJSONDropsUnknownFields(t *testing.T) {
	parser := NewV1BodyParser(testPolicy())
	var input PostBodyWriteInput
	err := json.Unmarshal([]byte(`{
		"schemaVersion": 1,
		"blocks": [
			{
				"type": "paragraph",
				"children": [{"type": "text", "text": "正文"}],
				"rawHtml": "<script>alert(1)</script>",
				"styleClass": "unknown"
			}
		]
	}`), &input)
	if err != nil {
		t.Fatalf("json.Unmarshal returned error: %v", err)
	}

	normalized, err := parser.Parse(context.Background(), input)

	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	canonical := string(normalized.CanonicalJSON)
	if strings.Contains(canonical, "rawHtml") || strings.Contains(canonical, "styleClass") {
		t.Fatalf("CanonicalJSON = %s, want unknown fields removed", canonical)
	}
	if !strings.Contains(canonical, `"schemaVersion":1`) || !strings.Contains(canonical, `"blocks"`) {
		t.Fatalf("CanonicalJSON = %s, want normalized body envelope", canonical)
	}
}

func TestV1BodyParserReportsUnknownJSONBlockAsValidationDetail(t *testing.T) {
	parser := NewV1BodyParser(testPolicy())
	var input PostBodyWriteInput
	err := json.Unmarshal([]byte(`{
		"schemaVersion": 1,
		"blocks": [{"type": "poll"}]
	}`), &input)
	if err != nil {
		t.Fatalf("json.Unmarshal returned error: %v", err)
	}

	_, err = parser.Parse(context.Background(), input)

	assertDetail(t, requireValidationError(t, err), "blocks[0].type", "BLOCK_TYPE_NOT_ENABLED")
}

func TestV1BodyParserRejectsMissingOrNullBlocks(t *testing.T) {
	parser := NewV1BodyParser(testPolicy())
	for _, tc := range []struct {
		name string
		raw  string
	}{
		{name: "missing", raw: `{"schemaVersion": 1}`},
		{name: "null", raw: `{"schemaVersion": 1, "blocks": null}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var input PostBodyWriteInput
			if err := json.Unmarshal([]byte(tc.raw), &input); err != nil {
				t.Fatalf("json.Unmarshal returned error: %v", err)
			}

			_, err := parser.Parse(context.Background(), input)

			assertDetail(t, requireValidationError(t, err), "blocks", "BODY_SCHEMA_INVALID")
		})
	}
}

func TestV1BodyParserRejectsTypedNilBlock(t *testing.T) {
	parser := NewV1BodyParser(testPolicy())
	var block *ParagraphBlock

	_, err := parser.Parse(context.Background(), PostBodyWriteInput{
		SchemaVersion: 1,
		Blocks:        Blocks{block},
	})

	assertDetail(t, requireValidationError(t, err), "blocks[0].type", "BLOCK_TYPE_REQUIRED")
}

func TestV1BodyParserRejectsUnsafeExternalInput(t *testing.T) {
	parser := NewV1BodyParser(testPolicy())

	_, err := parser.Parse(context.Background(), PostBodyWriteInput{
		SchemaVersion: 1,
		Blocks: Blocks{
			&ParagraphBlock{
				Children: []InlineNode{
					{
						Type: "text",
						Text: "危险链接",
						Marks: []InlineMark{
							{Type: "link", Href: "javascript:alert(1)"},
						},
					},
				},
			},
			&ExternalEmbedBlock{
				Provider: "iframe",
				URL:      "data:text/html,<script>alert(1)</script>",
			},
		},
	})

	validationErr := requireValidationError(t, err)
	assertDetail(t, validationErr, "blocks[0].children[0].marks[0].href", "LINK_HREF_UNSAFE")
	assertDetail(t, validationErr, "blocks[1].provider", "EXTERNAL_EMBED_PROVIDER_NOT_ALLOWED")
	assertDetail(t, validationErr, "blocks[1].url", "EXTERNAL_EMBED_URL_UNSAFE")
}

func TestV1BodyParserStopsCollectingErrorsAtPolicyLimit(t *testing.T) {
	policy := testPolicy()
	policy.MaxValidationErrors = 2
	parser := NewV1BodyParser(policy)

	_, err := parser.Parse(context.Background(), PostBodyWriteInput{
		SchemaVersion: 1,
		Blocks: Blocks{
			&UnsupportedBlock{Type: BlockType("unknown")},
			&UnsupportedBlock{Type: BlockType("unknown")},
			&UnsupportedBlock{Type: BlockType("unknown")},
		},
	})

	validationErr := requireValidationError(t, err)
	if len(validationErr.Details) != 2 {
		t.Fatalf("len(Details) = %d, want 2", len(validationErr.Details))
	}
	if !validationErr.Truncated {
		t.Fatalf("Truncated = false, want true")
	}
}

func TestV1BodyParserRejectsConfiguredLimits(t *testing.T) {
	t.Run("plain text length", func(t *testing.T) {
		policy := testPolicy()
		policy.MaxPlainTextChars = 3
		parser := NewV1BodyParser(policy)

		_, err := parser.Parse(context.Background(), PostBodyWriteInput{
			SchemaVersion: 1,
			Blocks: Blocks{
				&ParagraphBlock{
					Children: []InlineNode{
						{Type: "text", Text: "超过长度"},
					},
				},
			},
		})

		assertDetail(t, requireValidationError(t, err), "blocks", "BODY_TEXT_TOO_LONG")
	})

	t.Run("table cells", func(t *testing.T) {
		policy := testPolicy()
		policy.MaxTableCells = 1
		parser := NewV1BodyParser(policy)

		_, err := parser.Parse(context.Background(), PostBodyWriteInput{
			SchemaVersion: 1,
			Blocks: Blocks{
				&TableBlock{
					Headers: []TableCell{
						{Children: []InlineNode{{Type: "text", Text: "A"}}},
						{Children: []InlineNode{{Type: "text", Text: "B"}}},
					},
				},
			},
		})

		assertDetail(t, requireValidationError(t, err), "blocks[0]", "TABLE_TOO_LARGE")
	})
}

func TestV1BodyParserStopsTraversalAfterHardLimit(t *testing.T) {
	policy := testPolicy()
	policy.MaxInlineNodes = 1
	policy.MaxPlainTextChars = 8
	parser := NewV1BodyParser(policy)

	_, err := parser.Parse(context.Background(), PostBodyWriteInput{
		SchemaVersion: 1,
		Blocks: Blocks{
			&ParagraphBlock{
				Children: []InlineNode{
					{Type: "text", Text: "A"},
					{Type: "text", Text: "B"},
				},
			},
			&CodeBlock{Code: strings.Repeat("x", 64)},
		},
	})

	validationErr := requireValidationError(t, err)
	assertDetail(t, validationErr, "blocks", "BODY_INLINE_NODE_COUNT_EXCEEDED")
	assertNoDetail(t, validationErr, "blocks", "BODY_TEXT_TOO_LONG")
}

func requireValidationError(t *testing.T, err error) *BodyValidationError {
	t.Helper()
	if err == nil {
		t.Fatal("Parse returned nil error, want BodyValidationError")
	}
	validationErr, ok := err.(*BodyValidationError)
	if !ok {
		t.Fatalf("error = %T %v, want *BodyValidationError", err, err)
	}
	return validationErr
}

func assertNoDetail(t *testing.T, validationErr *BodyValidationError, path string, code string) {
	t.Helper()
	for _, detail := range validationErr.Details {
		if detail.Path == path && detail.Code == code {
			t.Fatalf("Details = %#v, unexpected path=%q code=%q", validationErr.Details, path, code)
		}
	}
}

func assertDetail(t *testing.T, validationErr *BodyValidationError, path string, code string) {
	t.Helper()
	for _, detail := range validationErr.Details {
		if detail.Path == path && detail.Code == code {
			return
		}
	}
	t.Fatalf("Details = %#v, missing path=%q code=%q", validationErr.Details, path, code)
}
