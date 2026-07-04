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
type BodyBlock = ports.BodyBlock
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
		Blocks: []BodyBlock{
			{
				"type": "paragraph",
				"children": []InlineNode{
					{
						"type": "text",
						"text": "安全链接",
						"marks": []InlineMark{
							{"type": "link", "href": "https://example.com/path?q=1"},
						},
					},
				},
			},
			{
				"type":   "image",
				"fileId": "file_123",
				"alt":    "示例图",
			},
			{
				"type":     "external_embed",
				"provider": "image",
				"url":      "https://cdn.example.com/image.png",
				"title":    "外部图",
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

func TestV1BodyParserCanonicalJSONDropsUnknownFields(t *testing.T) {
	parser := NewV1BodyParser(testPolicy())

	normalized, err := parser.Parse(context.Background(), PostBodyWriteInput{
		SchemaVersion: 1,
		Blocks: []BodyBlock{
			{
				"type":       "paragraph",
				"children":   []InlineNode{{"type": "text", "text": "正文"}},
				"rawHtml":    "<script>alert(1)</script>",
				"styleClass": "unknown",
			},
		},
	})

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

func TestV1BodyParserRejectsUnsafeExternalInput(t *testing.T) {
	parser := NewV1BodyParser(testPolicy())

	_, err := parser.Parse(context.Background(), PostBodyWriteInput{
		SchemaVersion: 1,
		Blocks: []BodyBlock{
			{
				"type": "paragraph",
				"children": []InlineNode{
					{
						"type": "text",
						"text": "危险链接",
						"marks": []InlineMark{
							{"type": "link", "href": "javascript:alert(1)"},
						},
					},
				},
			},
			{
				"type":     "external_embed",
				"provider": "iframe",
				"url":      "data:text/html,<script>alert(1)</script>",
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
		Blocks: []BodyBlock{
			{"type": "unknown"},
			{"type": "unknown"},
			{"type": "unknown"},
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
			Blocks: []BodyBlock{
				{
					"type": "paragraph",
					"children": []InlineNode{
						{"type": "text", "text": "超过长度"},
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
			Blocks: []BodyBlock{
				{
					"type": "table",
					"headers": []TableCell{
						{Children: []InlineNode{{"type": "text", "text": "A"}}},
						{Children: []InlineNode{{"type": "text", "text": "B"}}},
					},
				},
			},
		})

		assertDetail(t, requireValidationError(t, err), "blocks[0]", "TABLE_TOO_LARGE")
	})
}

func BenchmarkV1BodyParser(b *testing.B) {
	cases := []struct {
		name      string
		policy    BodyValidationPolicy
		input     PostBodyWriteInput
		wantError bool
	}{
		{
			name:  "small",
			input: benchmarkBody(20, strings.Repeat("小正文", 10)),
		},
		{
			name:  "medium",
			input: benchmarkBody(100, strings.Repeat("中等正文", 20)),
		},
		{
			name:  "near_limit",
			input: benchmarkBody(400, strings.Repeat("临界正文", 12)),
		},
		{
			name:  "many_blocks",
			input: benchmarkBody(1000, "段落"),
		},
		{
			name:  "large_table",
			input: benchmarkTableBody(1000),
		},
		{
			name:  "many_links",
			input: benchmarkLinksBody(200),
		},
		{
			name:  "large_code",
			input: benchmarkCodeBody(strings.Repeat("x", DefaultBodyValidationPolicy().MaxCodeBlockChars)),
		},
		{
			name: "reject_oversize",
			policy: func() BodyValidationPolicy {
				policy := DefaultBodyValidationPolicy()
				policy.MaxCanonicalJSONBytes = 128
				return policy
			}(),
			input:     benchmarkBody(4, strings.Repeat("oversize", 20)),
			wantError: true,
		},
		{
			name: "reject_many_errors",
			policy: func() BodyValidationPolicy {
				policy := DefaultBodyValidationPolicy()
				policy.MaxValidationErrors = 20
				return policy
			}(),
			input:     benchmarkUnknownBlocksBody(100),
			wantError: true,
		},
	}

	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			policy := tc.policy
			if policy == (BodyValidationPolicy{}) {
				policy = DefaultBodyValidationPolicy()
			}
			parser := NewV1BodyParser(policy)

			b.ReportAllocs()
			for index := 0; index < b.N; index++ {
				_, err := parser.Parse(context.Background(), tc.input)
				if tc.wantError {
					if err == nil {
						b.Fatal("Parse returned nil error, want validation error")
					}
					continue
				}
				if err != nil {
					b.Fatalf("Parse returned error: %v", err)
				}
			}
		})
	}
}

func benchmarkTableBody(cells int) PostBodyWriteInput {
	headers := make([]TableCell, 0, cells)
	for index := 0; index < cells; index++ {
		headers = append(headers, TableCell{
			Children: []InlineNode{{"type": "text", "text": "cell"}},
		})
	}
	return PostBodyWriteInput{
		SchemaVersion: 1,
		Blocks: []BodyBlock{
			{"type": "table", "headers": headers},
		},
	}
}

func benchmarkLinksBody(links int) PostBodyWriteInput {
	children := make([]InlineNode, 0, links)
	for index := 0; index < links; index++ {
		children = append(children, InlineNode{
			"type": "text",
			"text": "link",
			"marks": []InlineMark{
				{"type": "link", "href": "https://example.com"},
			},
		})
	}
	return PostBodyWriteInput{
		SchemaVersion: 1,
		Blocks: []BodyBlock{
			{"type": "paragraph", "children": children},
		},
	}
}

func benchmarkCodeBody(code string) PostBodyWriteInput {
	return PostBodyWriteInput{
		SchemaVersion: 1,
		Blocks: []BodyBlock{
			{"type": "code_block", "code": code},
		},
	}
}

func benchmarkUnknownBlocksBody(blocks int) PostBodyWriteInput {
	bodyBlocks := make([]BodyBlock, 0, blocks)
	for index := 0; index < blocks; index++ {
		bodyBlocks = append(bodyBlocks, BodyBlock{"type": "unknown"})
	}
	return PostBodyWriteInput{
		SchemaVersion: 1,
		Blocks:        bodyBlocks,
	}
}

func benchmarkBody(blocks int, text string) PostBodyWriteInput {
	bodyBlocks := make([]BodyBlock, 0, blocks)
	for index := 0; index < blocks; index++ {
		bodyBlocks = append(bodyBlocks, BodyBlock{
			"type": "paragraph",
			"children": []InlineNode{
				{"type": "text", "text": text},
			},
		})
	}
	return PostBodyWriteInput{
		SchemaVersion: 1,
		Blocks:        bodyBlocks,
	}
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

func assertDetail(t *testing.T, validationErr *BodyValidationError, path string, code string) {
	t.Helper()
	for _, detail := range validationErr.Details {
		if detail.Path == path && detail.Code == code {
			return
		}
	}
	t.Fatalf("Details = %#v, missing path=%q code=%q", validationErr.Details, path, code)
}
