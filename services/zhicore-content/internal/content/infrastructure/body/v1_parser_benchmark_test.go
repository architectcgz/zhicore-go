package body

import (
	"context"
	"strings"
	"testing"
)

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
			Children: []InlineNode{{Type: "text", Text: "cell"}},
		})
	}
	return PostBodyWriteInput{
		SchemaVersion: 1,
		Blocks: Blocks{
			&TableBlock{Headers: headers},
		},
	}
}

func benchmarkLinksBody(links int) PostBodyWriteInput {
	children := make([]InlineNode, 0, links)
	for index := 0; index < links; index++ {
		children = append(children, InlineNode{
			Type: "text",
			Text: "link",
			Marks: []InlineMark{
				{Type: "link", Href: "https://example.com"},
			},
		})
	}
	return PostBodyWriteInput{
		SchemaVersion: 1,
		Blocks: Blocks{
			&ParagraphBlock{Children: children},
		},
	}
}

func benchmarkCodeBody(code string) PostBodyWriteInput {
	return PostBodyWriteInput{
		SchemaVersion: 1,
		Blocks: Blocks{
			&CodeBlock{Code: code},
		},
	}
}

func benchmarkUnknownBlocksBody(blocks int) PostBodyWriteInput {
	bodyBlocks := make(Blocks, 0, blocks)
	for index := 0; index < blocks; index++ {
		bodyBlocks = append(bodyBlocks, &UnsupportedBlock{Type: BlockType("unknown")})
	}
	return PostBodyWriteInput{
		SchemaVersion: 1,
		Blocks:        bodyBlocks,
	}
}

func benchmarkBody(blocks int, text string) PostBodyWriteInput {
	bodyBlocks := make(Blocks, 0, blocks)
	for index := 0; index < blocks; index++ {
		bodyBlocks = append(bodyBlocks, &ParagraphBlock{
			Children: []InlineNode{
				{Type: "text", Text: text},
			},
		})
	}
	return PostBodyWriteInput{
		SchemaVersion: 1,
		Blocks:        bodyBlocks,
	}
}
