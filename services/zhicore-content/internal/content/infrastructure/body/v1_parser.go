// Package body owns Content body parsing and validation adapters.
package body

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"

	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
)

type V1BodyParser struct {
	policy ports.BodyValidationPolicy
}

func DefaultBodyValidationPolicy() ports.BodyValidationPolicy {
	return ports.BodyValidationPolicy{
		MaxCanonicalJSONBytes: 256 * 1024,
		MaxPlainTextChars:     20000,
		MaxBlocks:             1000,
		MaxInlineNodes:        5000,
		MaxContainerDepth:     2,
		MaxTableCells:         1000,
		MaxLatexChars:         3000,
		MaxCodeBlockChars:     20000,
		MaxExternalLinks:      200,
		MaxValidationErrors:   20,
	}
}

func NewV1BodyParser(policy ports.BodyValidationPolicy) *V1BodyParser {
	defaults := DefaultBodyValidationPolicy()
	if policy.MaxCanonicalJSONBytes == 0 {
		policy.MaxCanonicalJSONBytes = defaults.MaxCanonicalJSONBytes
	}
	if policy.MaxPlainTextChars == 0 {
		policy.MaxPlainTextChars = defaults.MaxPlainTextChars
	}
	if policy.MaxBlocks == 0 {
		policy.MaxBlocks = defaults.MaxBlocks
	}
	if policy.MaxInlineNodes == 0 {
		policy.MaxInlineNodes = defaults.MaxInlineNodes
	}
	if policy.MaxContainerDepth == 0 {
		policy.MaxContainerDepth = defaults.MaxContainerDepth
	}
	if policy.MaxTableCells == 0 {
		policy.MaxTableCells = defaults.MaxTableCells
	}
	if policy.MaxLatexChars == 0 {
		policy.MaxLatexChars = defaults.MaxLatexChars
	}
	if policy.MaxCodeBlockChars == 0 {
		policy.MaxCodeBlockChars = defaults.MaxCodeBlockChars
	}
	if policy.MaxExternalLinks == 0 {
		policy.MaxExternalLinks = defaults.MaxExternalLinks
	}
	if policy.MaxValidationErrors == 0 {
		policy.MaxValidationErrors = defaults.MaxValidationErrors
	}
	return &V1BodyParser{policy: policy}
}

func (p *V1BodyParser) Parse(ctx context.Context, input ports.PostBodyWriteInput) (ports.NormalizedBody, error) {
	if err := ctx.Err(); err != nil {
		return ports.NormalizedBody{}, err
	}

	blocksPath := rootPath("blocks")
	state := parserState{
		policy:          p.policy,
		plainTextBlocks: make([]string, 0, min(len(input.Blocks), p.policy.MaxBlocks)),
		mediaRefs:       make([]ports.MediaRef, 0, 4),
		externalLinks:   make([]string, 0, 4),
	}

	if input.SchemaVersion != 1 {
		state.addError(rootPath("schemaVersion"), "BODY_SCHEMA_UNSUPPORTED")
	}
	if input.Blocks == nil {
		// SaveDraftBody 的 contract 要求 blocks 必填；显式空正文必须传 []，
		// 避免缺失或 null 被 canonicalize 成合法空正文。
		state.addError(blocksPath, "BODY_SCHEMA_INVALID")
	}

	state.validateBlocks(input.Blocks, blocksPath, 0)
	plainText := strings.Join(state.plainTextBlocks, "\n")
	if len([]rune(plainText)) > p.policy.MaxPlainTextChars {
		state.addError(blocksPath, "BODY_TEXT_TOO_LONG")
	}

	if len(state.details) > 0 {
		return ports.NormalizedBody{}, &ports.BodyValidationError{
			Details:   state.details,
			Truncated: state.truncated,
		}
	}

	canonicalJSON, err := json.Marshal(canonicalizeBody(input))
	if err != nil {
		return ports.NormalizedBody{}, err
	}
	if len(canonicalJSON) > p.policy.MaxCanonicalJSONBytes {
		return ports.NormalizedBody{}, &ports.BodyValidationError{
			Details: []ports.ValidationDetail{
				{Path: "blocks", Code: "BODY_TOO_LARGE"},
			},
		}
	}

	hash := sha256.Sum256(canonicalJSON)
	return ports.NormalizedBody{
		PlainText:     plainText,
		MediaRefs:     state.mediaRefs,
		ExternalLinks: state.externalLinks,
		CanonicalJSON: canonicalJSON,
		ContentHash:   "sha256:" + hex.EncodeToString(hash[:]),
		SizeBytes:     len(canonicalJSON),
		BlockCount:    state.blockCount,
	}, nil
}

type canonicalBody struct {
	SchemaVersion int              `json:"schemaVersion"`
	Blocks        []canonicalBlock `json:"blocks"`
}

type canonicalBlock interface {
	canonicalBlock()
}

type canonicalParagraphBlock struct {
	Type     ports.BlockType       `json:"type"`
	Children []canonicalInlineNode `json:"children,omitempty"`
}

func (canonicalParagraphBlock) canonicalBlock() {}

type canonicalHeadingBlock struct {
	Type     ports.BlockType       `json:"type"`
	Level    int                   `json:"level,omitempty"`
	Children []canonicalInlineNode `json:"children,omitempty"`
}

func (canonicalHeadingBlock) canonicalBlock() {}

type canonicalQuoteBlock struct {
	Type   ports.BlockType  `json:"type"`
	Blocks []canonicalBlock `json:"blocks,omitempty"`
}

func (canonicalQuoteBlock) canonicalBlock() {}

type canonicalListBlock struct {
	Type  ports.BlockType     `json:"type"`
	Items []canonicalListItem `json:"items,omitempty"`
}

func (canonicalListBlock) canonicalBlock() {}

type canonicalListItem struct {
	Blocks []canonicalBlock `json:"blocks,omitempty"`
}

type canonicalCodeBlock struct {
	Type     ports.BlockType `json:"type"`
	Code     string          `json:"code"`
	Language string          `json:"language,omitempty"`
}

func (canonicalCodeBlock) canonicalBlock() {}

type canonicalTableBlock struct {
	Type    ports.BlockType        `json:"type"`
	Headers []canonicalTableCell   `json:"headers,omitempty"`
	Rows    [][]canonicalTableCell `json:"rows,omitempty"`
}

func (canonicalTableBlock) canonicalBlock() {}

type canonicalTableCell struct {
	Children []canonicalInlineNode `json:"children"`
}

type canonicalCollapsibleBlock struct {
	Type   ports.BlockType       `json:"type"`
	Title  []canonicalInlineNode `json:"title,omitempty"`
	Blocks []canonicalBlock      `json:"blocks,omitempty"`
}

func (canonicalCollapsibleBlock) canonicalBlock() {}

type canonicalMathBlock struct {
	Type  ports.BlockType `json:"type"`
	Latex string          `json:"latex"`
}

func (canonicalMathBlock) canonicalBlock() {}

type canonicalImageBlock struct {
	Type   ports.BlockType `json:"type"`
	FileID string          `json:"fileId"`
	Alt    string          `json:"alt,omitempty"`
}

func (canonicalImageBlock) canonicalBlock() {}

type canonicalExternalEmbedBlock struct {
	Type     ports.BlockType `json:"type"`
	Provider string          `json:"provider"`
	URL      string          `json:"url,omitempty"`
	Title    string          `json:"title,omitempty"`
}

func (canonicalExternalEmbedBlock) canonicalBlock() {}

type canonicalAttachmentGalleryBlock struct {
	Type  ports.BlockType           `json:"type"`
	Items []canonicalAttachmentItem `json:"items,omitempty"`
}

func (canonicalAttachmentGalleryBlock) canonicalBlock() {}

type canonicalAttachmentItem struct {
	FileID string `json:"fileId"`
}

type canonicalInlineNode struct {
	Type  string                `json:"type"`
	Text  string                `json:"text,omitempty"`
	Marks []canonicalInlineMark `json:"marks,omitempty"`
}

type canonicalInlineMark struct {
	Type string `json:"type"`
	Href string `json:"href,omitempty"`
}

func canonicalizeBody(input ports.PostBodyWriteInput) canonicalBody {
	return canonicalBody{
		SchemaVersion: input.SchemaVersion,
		Blocks:        canonicalizeBlocks(input.Blocks),
	}
}

func canonicalizeBlocks(blocks ports.Blocks) []canonicalBlock {
	canonicalBlocks := make([]canonicalBlock, 0, len(blocks))
	for _, block := range blocks {
		canonicalBlocks = append(canonicalBlocks, canonicalizeBlock(block))
	}
	return canonicalBlocks
}

func canonicalizeBlock(block ports.Block) canonicalBlock {
	switch typed := block.(type) {
	case *ports.ParagraphBlock:
		return canonicalParagraphBlock{Type: ports.BlockParagraph, Children: canonicalizeInlineNodes(typed.Children)}
	case *ports.HeadingBlock:
		return canonicalHeadingBlock{Type: ports.BlockHeading, Level: typed.Level, Children: canonicalizeInlineNodes(typed.Children)}
	case *ports.QuoteBlock:
		return canonicalQuoteBlock{Type: ports.BlockQuote, Blocks: canonicalizeBlocks(typed.Blocks)}
	case *ports.ListBlock:
		return canonicalListBlock{Type: ports.BlockList, Items: canonicalizeListItems(typed.Items)}
	case *ports.CodeBlock:
		return canonicalCodeBlock{Type: ports.BlockCode, Code: typed.Code, Language: typed.Language}
	case *ports.TableBlock:
		return canonicalTableBlock{Type: ports.BlockTable, Headers: canonicalizeTableCells(typed.Headers), Rows: canonicalizeTableRows(typed.Rows)}
	case *ports.CollapsibleBlock:
		return canonicalCollapsibleBlock{Type: ports.BlockCollapsible, Title: canonicalizeInlineNodes(typed.Title), Blocks: canonicalizeBlocks(typed.Blocks)}
	case *ports.MathBlock:
		return canonicalMathBlock{Type: ports.BlockMath, Latex: typed.Latex}
	case *ports.ImageBlock:
		return canonicalImageBlock{Type: ports.BlockImage, FileID: strings.TrimSpace(typed.FileID), Alt: typed.Alt}
	case *ports.ExternalEmbedBlock:
		safeURL, _ := sanitizeHTTPURL(typed.URL)
		return canonicalExternalEmbedBlock{Type: ports.BlockExternalEmbed, Provider: typed.Provider, URL: safeURL, Title: typed.Title}
	case *ports.AttachmentGalleryBlock:
		return canonicalAttachmentGalleryBlock{Type: ports.BlockAttachmentGallery, Items: canonicalizeAttachmentItems(typed.Items)}
	default:
		return canonicalParagraphBlock{Type: ""}
	}
}

func canonicalizeListItems(items []ports.ListItem) []canonicalListItem {
	canonicalItems := make([]canonicalListItem, 0, len(items))
	for _, item := range items {
		canonicalItems = append(canonicalItems, canonicalListItem{Blocks: canonicalizeBlocks(item.Blocks)})
	}
	return canonicalItems
}

func canonicalizeInlineNodes(nodes []ports.InlineNode) []canonicalInlineNode {
	canonicalNodes := make([]canonicalInlineNode, 0, len(nodes))
	for _, node := range nodes {
		canonicalNodes = append(canonicalNodes, canonicalInlineNode{
			Type:  "text",
			Text:  node.Text,
			Marks: canonicalizeMarks(node.Marks),
		})
	}
	return canonicalNodes
}

func canonicalizeMarks(marks []ports.InlineMark) []canonicalInlineMark {
	canonicalMarks := make([]canonicalInlineMark, 0, len(marks))
	for _, mark := range marks {
		canonical := canonicalInlineMark{Type: mark.Type}
		if mark.Type == "link" {
			if safeURL, ok := sanitizeHTTPURL(mark.Href); ok {
				canonical.Href = safeURL
			}
		}
		canonicalMarks = append(canonicalMarks, canonical)
	}
	return canonicalMarks
}

func canonicalizeTableCells(cells []ports.TableCell) []canonicalTableCell {
	canonicalCells := make([]canonicalTableCell, 0, len(cells))
	for _, cell := range cells {
		canonicalCells = append(canonicalCells, canonicalTableCell{
			Children: canonicalizeInlineNodes(cell.Children),
		})
	}
	return canonicalCells
}

func canonicalizeTableRows(rows [][]ports.TableCell) [][]canonicalTableCell {
	canonicalRows := make([][]canonicalTableCell, 0, len(rows))
	for _, row := range rows {
		canonicalRows = append(canonicalRows, canonicalizeTableCells(row))
	}
	return canonicalRows
}

func canonicalizeAttachmentItems(items []ports.AttachmentItem) []canonicalAttachmentItem {
	canonicalItems := make([]canonicalAttachmentItem, 0, len(items))
	for _, item := range items {
		canonicalItems = append(canonicalItems, canonicalAttachmentItem{
			FileID: strings.TrimSpace(item.FileID),
		})
	}
	return canonicalItems
}

type parserState struct {
	policy          ports.BodyValidationPolicy
	details         []ports.ValidationDetail
	truncated       bool
	stopValidation  bool
	blockCount      int
	inlineNodeCount int
	tableCellCount  int
	plainTextBlocks []string
	mediaRefs       []ports.MediaRef
	externalLinks   []string
}

func (s *parserState) addError(path []pathPart, code string) {
	if len(s.details) >= s.policy.MaxValidationErrors {
		s.truncated = true
		s.stopValidation = true
		return
	}
	s.details = append(s.details, ports.ValidationDetail{Path: formatPath(path), Code: code})
	if len(s.details) >= s.policy.MaxValidationErrors {
		s.truncated = true
		s.stopValidation = true
	}
}

func (s *parserState) stopAfterHardLimit() {
	s.stopValidation = true
}

func (s *parserState) validateBlocks(blocks ports.Blocks, path []pathPart, containerDepth int) {
	for index, block := range blocks {
		if s.stopValidation {
			return
		}
		s.validateBlock(block, indexedPath(path, index), containerDepth)
	}
}

func (s *parserState) validateBlock(block ports.Block, path []pathPart, containerDepth int) {
	s.blockCount++
	if s.blockCount > s.policy.MaxBlocks {
		s.addError(rootPath("blocks"), "BODY_BLOCK_COUNT_EXCEEDED")
		s.stopAfterHardLimit()
		return
	}

	switch typed := block.(type) {
	case nil:
		s.addError(fieldPath(path, "type"), "BLOCK_TYPE_REQUIRED")
	case *ports.UnsupportedBlock:
		// Unknown JSON block types are decoded into UnsupportedBlock so parser,
		// not JSON decoding, remains the owner of field-level validation errors.
		if typed == nil {
			s.addError(fieldPath(path, "type"), "BLOCK_TYPE_REQUIRED")
			return
		}
		if typed.Type == "" {
			s.addError(fieldPath(path, "type"), "BLOCK_TYPE_REQUIRED")
			return
		}
		s.addError(fieldPath(path, "type"), "BLOCK_TYPE_NOT_ENABLED")
	case *ports.ParagraphBlock:
		if typed == nil {
			s.addError(fieldPath(path, "type"), "BLOCK_TYPE_REQUIRED")
			return
		}
		text := s.validateInlineNodes(typed.Children, fieldPath(path, "children"))
		s.appendPlainText(text)
	case *ports.HeadingBlock:
		if typed == nil {
			s.addError(fieldPath(path, "type"), "BLOCK_TYPE_REQUIRED")
			return
		}
		text := s.validateInlineNodes(typed.Children, fieldPath(path, "children"))
		s.appendPlainText(text)
	case *ports.QuoteBlock:
		if typed == nil {
			s.addError(fieldPath(path, "type"), "BLOCK_TYPE_REQUIRED")
			return
		}
		s.validateContainerDepth(path, containerDepth)
		if s.stopValidation {
			return
		}
		s.validateBlocks(typed.Blocks, fieldPath(path, "blocks"), containerDepth+1)
	case *ports.ListBlock:
		if typed == nil {
			s.addError(fieldPath(path, "type"), "BLOCK_TYPE_REQUIRED")
			return
		}
		s.validateContainerDepth(path, containerDepth)
		if s.stopValidation {
			return
		}
		s.validateListItems(typed.Items, fieldPath(path, "items"), containerDepth+1)
	case *ports.CollapsibleBlock:
		if typed == nil {
			s.addError(fieldPath(path, "type"), "BLOCK_TYPE_REQUIRED")
			return
		}
		s.validateContainerDepth(path, containerDepth)
		if s.stopValidation {
			return
		}
		s.validateInlineNodes(typed.Title, fieldPath(path, "title"))
		if s.stopValidation {
			return
		}
		s.validateBlocks(typed.Blocks, fieldPath(path, "blocks"), containerDepth+1)
	case *ports.CodeBlock:
		if typed == nil {
			s.addError(fieldPath(path, "type"), "BLOCK_TYPE_REQUIRED")
			return
		}
		if len([]rune(typed.Code)) > s.policy.MaxCodeBlockChars {
			s.addError(fieldPath(path, "code"), "CODE_BLOCK_TOO_LONG")
		}
		s.appendPlainText(typed.Code)
	case *ports.MathBlock:
		if typed == nil {
			s.addError(fieldPath(path, "type"), "BLOCK_TYPE_REQUIRED")
			return
		}
		if len([]rune(typed.Latex)) > s.policy.MaxLatexChars {
			s.addError(fieldPath(path, "latex"), "MATH_LATEX_TOO_LONG")
		}
		s.appendPlainText(typed.Latex)
	case *ports.TableBlock:
		if typed == nil {
			s.addError(fieldPath(path, "type"), "BLOCK_TYPE_REQUIRED")
			return
		}
		s.validateTable(typed, path)
	case *ports.ImageBlock:
		if typed == nil {
			s.addError(fieldPath(path, "type"), "BLOCK_TYPE_REQUIRED")
			return
		}
		fileID := strings.TrimSpace(typed.FileID)
		if fileID == "" {
			s.addError(fieldPath(path, "fileId"), "MEDIA_REF_INVALID")
			return
		}
		s.mediaRefs = append(s.mediaRefs, ports.MediaRef{FileID: fileID})
	case *ports.ExternalEmbedBlock:
		if typed == nil {
			s.addError(fieldPath(path, "type"), "BLOCK_TYPE_REQUIRED")
			return
		}
		s.validateExternalEmbed(typed, path)
	case *ports.AttachmentGalleryBlock:
		if typed == nil {
			s.addError(fieldPath(path, "type"), "BLOCK_TYPE_REQUIRED")
			return
		}
		s.validateAttachmentGallery(typed.Items, fieldPath(path, "items"))
	default:
		s.addError(fieldPath(path, "type"), "BLOCK_TYPE_NOT_ENABLED")
	}
}

func (s *parserState) validateContainerDepth(path []pathPart, parentDepth int) {
	if parentDepth+1 > s.policy.MaxContainerDepth {
		s.addError(path, "BODY_CONTAINER_TOO_DEEP")
		s.stopAfterHardLimit()
	}
}

func (s *parserState) validateInlineNodes(nodes []ports.InlineNode, path []pathPart) string {
	var textBuilder strings.Builder
	for index, node := range nodes {
		if s.stopValidation {
			return textBuilder.String()
		}
		nodePath := indexedPath(path, index)
		s.inlineNodeCount++
		if s.inlineNodeCount > s.policy.MaxInlineNodes {
			s.addError(rootPath("blocks"), "BODY_INLINE_NODE_COUNT_EXCEEDED")
			s.stopAfterHardLimit()
			return textBuilder.String()
		}

		if node.Type != "text" {
			s.addError(fieldPath(nodePath, "type"), "INLINE_NODE_UNSUPPORTED")
			continue
		}

		textBuilder.WriteString(node.Text)
		s.validateMarks(node.Marks, fieldPath(nodePath, "marks"))
	}
	return textBuilder.String()
}

func (s *parserState) validateMarks(marks []ports.InlineMark, path []pathPart) {
	for index, mark := range marks {
		if s.stopValidation {
			return
		}
		markPath := indexedPath(path, index)
		if mark.Type == "" {
			s.addError(fieldPath(markPath, "type"), "INLINE_MARK_TYPE_REQUIRED")
			continue
		}
		switch mark.Type {
		case "bold", "italic", "underline", "strike", "inline_code":
		case "link":
			safeURL, ok := sanitizeHTTPURL(mark.Href)
			if !ok {
				s.addError(fieldPath(markPath, "href"), "LINK_HREF_UNSAFE")
				continue
			}
			s.addExternalLink(fieldPath(markPath, "href"), safeURL)
		default:
			s.addError(fieldPath(markPath, "type"), "INLINE_MARK_UNSUPPORTED")
		}
	}
}

func (s *parserState) validateListItems(items []ports.ListItem, path []pathPart, containerDepth int) {
	if items == nil {
		s.addError(path, "LIST_ITEMS_INVALID")
		return
	}
	for index, item := range items {
		if s.stopValidation {
			return
		}
		s.validateBlocks(item.Blocks, fieldPath(indexedPath(path, index), "blocks"), containerDepth)
	}
}

func (s *parserState) validateTable(block *ports.TableBlock, path []pathPart) {
	cellCount := len(block.Headers)
	for _, row := range block.Rows {
		cellCount += len(row)
	}
	s.tableCellCount += cellCount
	if s.tableCellCount > s.policy.MaxTableCells {
		s.addError(path, "TABLE_TOO_LARGE")
		s.stopAfterHardLimit()
		return
	}
	for index, cell := range block.Headers {
		if s.stopValidation {
			return
		}
		s.validateInlineNodes(cell.Children, fieldPath(indexedFieldPath(path, "headers", index), "children"))
	}
	for rowIndex, row := range block.Rows {
		if s.stopValidation {
			return
		}
		for cellIndex, cell := range row {
			if s.stopValidation {
				return
			}
			s.validateInlineNodes(cell.Children, fieldPath(indexedFieldPath2(path, "rows", rowIndex, cellIndex), "children"))
		}
	}
}

func (s *parserState) validateExternalEmbed(block *ports.ExternalEmbedBlock, path []pathPart) {
	if block.Provider != "image" {
		s.addError(fieldPath(path, "provider"), "EXTERNAL_EMBED_PROVIDER_NOT_ALLOWED")
	}
	safeURL, ok := sanitizeHTTPURL(block.URL)
	if !ok {
		s.addError(fieldPath(path, "url"), "EXTERNAL_EMBED_URL_UNSAFE")
		return
	}
	s.addExternalLink(fieldPath(path, "url"), safeURL)
}

func (s *parserState) validateAttachmentGallery(items []ports.AttachmentItem, path []pathPart) {
	if items == nil {
		s.addError(path, "ATTACHMENT_GALLERY_ITEMS_INVALID")
		return
	}
	for index, item := range items {
		fileID := strings.TrimSpace(item.FileID)
		if fileID == "" {
			s.addError(fieldPath(indexedPath(path, index), "fileId"), "MEDIA_REF_INVALID")
			continue
		}
		s.mediaRefs = append(s.mediaRefs, ports.MediaRef{FileID: fileID})
	}
}

func (s *parserState) appendPlainText(text string) {
	if text != "" {
		s.plainTextBlocks = append(s.plainTextBlocks, text)
	}
}

func (s *parserState) addExternalLink(path []pathPart, link string) {
	if len(s.externalLinks) >= s.policy.MaxExternalLinks {
		s.addError(path, "BODY_EXTERNAL_LINK_COUNT_EXCEEDED")
		s.stopAfterHardLimit()
		return
	}
	s.externalLinks = append(s.externalLinks, link)
}

func sanitizeHTTPURL(rawURL string) (string, bool) {
	parsedURL, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return "", false
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return "", false
	}
	return parsedURL.String(), true
}

type pathPart struct {
	field      string
	index0     int
	index1     int
	indexCount int
}

func rootPath(field string) []pathPart {
	parts := make([]pathPart, 1, 16)
	parts[0] = pathPart{field: field}
	return parts
}

func fieldPath(path []pathPart, field string) []pathPart {
	return append(path, pathPart{field: field})
}

func indexedPath(path []pathPart, index int) []pathPart {
	return append(path, pathPart{index0: index, indexCount: 1})
}

func indexedFieldPath(path []pathPart, field string, index int) []pathPart {
	return append(path, pathPart{field: field, index0: index, indexCount: 1})
}

func indexedFieldPath2(path []pathPart, field string, index0 int, index1 int) []pathPart {
	return append(path, pathPart{field: field, index0: index0, index1: index1, indexCount: 2})
}

func formatPath(path []pathPart) string {
	var builder strings.Builder
	for index, part := range path {
		if part.field != "" {
			if index > 0 {
				builder.WriteByte('.')
			}
			builder.WriteString(part.field)
		}
		if part.indexCount >= 1 {
			builder.WriteByte('[')
			builder.WriteString(strconv.Itoa(part.index0))
			builder.WriteByte(']')
		}
		if part.indexCount >= 2 {
			builder.WriteByte('[')
			builder.WriteString(strconv.Itoa(part.index1))
			builder.WriteByte(']')
		}
	}
	return builder.String()
}
