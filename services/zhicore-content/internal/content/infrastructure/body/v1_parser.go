// Package body owns Content body parsing and validation adapters.
package body

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
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

	state := parserState{
		policy: p.policy,
	}

	if input.SchemaVersion != 1 {
		state.addError("schemaVersion", "BODY_SCHEMA_UNSUPPORTED")
	}

	state.validateBlocks(input.Blocks, "blocks", 0)
	plainText := strings.Join(state.plainTextBlocks, "\n")
	if len([]rune(plainText)) > p.policy.MaxPlainTextChars {
		state.addError("blocks", "BODY_TEXT_TOO_LONG")
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
	Blocks        []map[string]any `json:"blocks"`
}

func canonicalizeBody(input ports.PostBodyWriteInput) canonicalBody {
	return canonicalBody{
		SchemaVersion: input.SchemaVersion,
		Blocks:        canonicalizeBlocks(input.Blocks),
	}
}

func canonicalizeBlocks(blocks []ports.BodyBlock) []map[string]any {
	canonicalBlocks := make([]map[string]any, 0, len(blocks))
	for _, block := range blocks {
		canonicalBlocks = append(canonicalBlocks, canonicalizeBlock(block))
	}
	return canonicalBlocks
}

func canonicalizeBlock(block ports.BodyBlock) map[string]any {
	blockType, _ := stringField(block, "type")
	canonical := map[string]any{"type": blockType}
	switch blockType {
	case "paragraph", "heading":
		canonical["children"] = canonicalizeInlineNodes(inlineNodesField(block, "children"))
		if level, ok := block["level"].(float64); ok {
			canonical["level"] = level
		}
	case "quote":
		canonical["blocks"] = canonicalizeBlocks(blocksField(block, "blocks"))
	case "list":
		canonical["items"] = canonicalizeListItems(block["items"])
	case "collapsible":
		canonical["title"] = canonicalizeInlineNodes(inlineNodesField(block, "title"))
		canonical["blocks"] = canonicalizeBlocks(blocksField(block, "blocks"))
	case "code_block":
		code, _ := stringField(block, "code")
		canonical["code"] = code
		if language, ok := stringField(block, "language"); ok {
			canonical["language"] = language
		}
	case "math":
		latex, _ := stringField(block, "latex")
		canonical["latex"] = latex
	case "table":
		canonical["headers"] = canonicalizeTableCells(tableCellsField(block, "headers"))
		canonical["rows"] = canonicalizeTableRows(tableRowsField(block, "rows"))
	case "image":
		fileID, _ := stringField(block, "fileId")
		canonical["fileId"] = strings.TrimSpace(fileID)
		if alt, ok := stringField(block, "alt"); ok {
			canonical["alt"] = alt
		}
	case "external_embed":
		provider, _ := stringField(block, "provider")
		canonical["provider"] = provider
		rawURL, _ := stringField(block, "url")
		if safeURL, ok := sanitizeHTTPURL(rawURL); ok {
			canonical["url"] = safeURL
		}
		if title, ok := stringField(block, "title"); ok {
			canonical["title"] = title
		}
	case "attachment_gallery":
		canonical["items"] = canonicalizeAttachmentItems(block["items"])
	}
	return canonical
}

func canonicalizeInlineNodes(nodes []ports.InlineNode) []map[string]any {
	canonicalNodes := make([]map[string]any, 0, len(nodes))
	for _, node := range nodes {
		text, _ := stringField(node, "text")
		canonicalNodes = append(canonicalNodes, map[string]any{
			"type":  "text",
			"text":  text,
			"marks": canonicalizeMarks(marksField(node, "marks")),
		})
	}
	return canonicalNodes
}

func canonicalizeMarks(marks []ports.InlineMark) []map[string]any {
	canonicalMarks := make([]map[string]any, 0, len(marks))
	for _, mark := range marks {
		markType, _ := stringField(mark, "type")
		canonical := map[string]any{"type": markType}
		if markType == "link" {
			href, _ := stringField(mark, "href")
			if safeURL, ok := sanitizeHTTPURL(href); ok {
				canonical["href"] = safeURL
			}
		}
		canonicalMarks = append(canonicalMarks, canonical)
	}
	return canonicalMarks
}

func canonicalizeListItems(value any) []map[string]any {
	items, ok := mapItems(value)
	if !ok {
		return nil
	}
	canonicalItems := make([]map[string]any, 0, len(items))
	for _, item := range items {
		canonicalItems = append(canonicalItems, map[string]any{
			"blocks": canonicalizeBlocks(blocksField(item, "blocks")),
		})
	}
	return canonicalItems
}

func canonicalizeTableCells(cells []ports.TableCell) []map[string]any {
	canonicalCells := make([]map[string]any, 0, len(cells))
	for _, cell := range cells {
		canonicalCells = append(canonicalCells, map[string]any{
			"children": canonicalizeInlineNodes(cell.Children),
		})
	}
	return canonicalCells
}

func canonicalizeTableRows(rows [][]ports.TableCell) [][]map[string]any {
	canonicalRows := make([][]map[string]any, 0, len(rows))
	for _, row := range rows {
		canonicalRows = append(canonicalRows, canonicalizeTableCells(row))
	}
	return canonicalRows
}

func canonicalizeAttachmentItems(value any) []map[string]any {
	items, ok := mapItems(value)
	if !ok {
		return nil
	}
	canonicalItems := make([]map[string]any, 0, len(items))
	for _, item := range items {
		fileID, _ := stringField(item, "fileId")
		canonicalItems = append(canonicalItems, map[string]any{
			"fileId": strings.TrimSpace(fileID),
		})
	}
	return canonicalItems
}

type parserState struct {
	policy          ports.BodyValidationPolicy
	details         []ports.ValidationDetail
	truncated       bool
	blockCount      int
	inlineNodeCount int
	tableCellCount  int
	plainTextBlocks []string
	mediaRefs       []ports.MediaRef
	externalLinks   []string
}

func (s *parserState) addError(path string, code string) {
	if len(s.details) >= s.policy.MaxValidationErrors {
		s.truncated = true
		return
	}
	s.details = append(s.details, ports.ValidationDetail{Path: path, Code: code})
}

func (s *parserState) validateBlocks(blocks []ports.BodyBlock, path string, containerDepth int) {
	for index, block := range blocks {
		blockPath := fmt.Sprintf("%s[%d]", path, index)
		s.validateBlock(block, blockPath, containerDepth)
	}
}

func (s *parserState) validateBlock(block ports.BodyBlock, path string, containerDepth int) {
	s.blockCount++
	if s.blockCount > s.policy.MaxBlocks {
		s.addError("blocks", "BODY_BLOCK_COUNT_EXCEEDED")
		return
	}

	blockType, ok := stringField(block, "type")
	if !ok {
		s.addError(path+".type", "BLOCK_TYPE_REQUIRED")
		return
	}

	switch blockType {
	case "paragraph":
		text := s.validateInlineNodes(inlineNodesField(block, "children"), path+".children")
		s.appendPlainText(text)
	case "heading":
		text := s.validateInlineNodes(inlineNodesField(block, "children"), path+".children")
		s.appendPlainText(text)
	case "quote":
		s.validateContainerDepth(path, containerDepth)
		s.validateBlocks(blocksField(block, "blocks"), path+".blocks", containerDepth+1)
	case "list":
		s.validateContainerDepth(path, containerDepth)
		s.validateListItems(block["items"], path+".items", containerDepth+1)
	case "collapsible":
		s.validateContainerDepth(path, containerDepth)
		s.validateInlineNodes(inlineNodesField(block, "title"), path+".title")
		s.validateBlocks(blocksField(block, "blocks"), path+".blocks", containerDepth+1)
	case "code_block":
		code, _ := stringField(block, "code")
		if len([]rune(code)) > s.policy.MaxCodeBlockChars {
			s.addError(path+".code", "CODE_BLOCK_TOO_LONG")
		}
		s.appendPlainText(code)
	case "math":
		latex, _ := stringField(block, "latex")
		if len([]rune(latex)) > s.policy.MaxLatexChars {
			s.addError(path+".latex", "MATH_LATEX_TOO_LONG")
		}
		s.appendPlainText(latex)
	case "table":
		s.validateTable(block, path)
	case "image":
		fileID, ok := stringField(block, "fileId")
		if !ok || strings.TrimSpace(fileID) == "" {
			s.addError(path+".fileId", "MEDIA_REF_INVALID")
			return
		}
		s.mediaRefs = append(s.mediaRefs, ports.MediaRef{FileID: strings.TrimSpace(fileID)})
	case "external_embed":
		s.validateExternalEmbed(block, path)
	case "attachment_gallery":
		s.validateAttachmentGallery(block["items"], path+".items")
	default:
		s.addError(path+".type", "BLOCK_TYPE_NOT_ENABLED")
	}
}

func (s *parserState) validateContainerDepth(path string, parentDepth int) {
	if parentDepth+1 > s.policy.MaxContainerDepth {
		s.addError(path, "BODY_CONTAINER_TOO_DEEP")
	}
}

func (s *parserState) validateInlineNodes(nodes []ports.InlineNode, path string) string {
	var textBuilder strings.Builder
	for index, node := range nodes {
		nodePath := fmt.Sprintf("%s[%d]", path, index)
		s.inlineNodeCount++
		if s.inlineNodeCount > s.policy.MaxInlineNodes {
			s.addError("blocks", "BODY_INLINE_NODE_COUNT_EXCEEDED")
			return textBuilder.String()
		}

		nodeType, ok := stringField(node, "type")
		if !ok || nodeType != "text" {
			s.addError(nodePath+".type", "INLINE_NODE_UNSUPPORTED")
			continue
		}

		text, _ := stringField(node, "text")
		textBuilder.WriteString(text)
		s.validateMarks(marksField(node, "marks"), nodePath+".marks")
	}
	return textBuilder.String()
}

func (s *parserState) validateMarks(marks []ports.InlineMark, path string) {
	for index, mark := range marks {
		markPath := fmt.Sprintf("%s[%d]", path, index)
		markType, ok := stringField(mark, "type")
		if !ok {
			s.addError(markPath+".type", "INLINE_MARK_TYPE_REQUIRED")
			continue
		}
		switch markType {
		case "bold", "italic", "underline", "strike", "inline_code":
		case "link":
			href, _ := stringField(mark, "href")
			safeURL, ok := sanitizeHTTPURL(href)
			if !ok {
				s.addError(markPath+".href", "LINK_HREF_UNSAFE")
				continue
			}
			s.addExternalLink(markPath+".href", safeURL)
		default:
			s.addError(markPath+".type", "INLINE_MARK_UNSUPPORTED")
		}
	}
}

func (s *parserState) validateListItems(value any, path string, containerDepth int) {
	items, ok := mapItems(value)
	if !ok {
		s.addError(path, "LIST_ITEMS_INVALID")
		return
	}
	for index, item := range items {
		s.validateBlocks(blocksField(item, "blocks"), fmt.Sprintf("%s[%d].blocks", path, index), containerDepth)
	}
}

func (s *parserState) validateTable(block ports.BodyBlock, path string) {
	headers := tableCellsField(block, "headers")
	rows := tableRowsField(block, "rows")
	cellCount := len(headers)
	for _, row := range rows {
		cellCount += len(row)
	}
	s.tableCellCount += cellCount
	if s.tableCellCount > s.policy.MaxTableCells {
		s.addError(path, "TABLE_TOO_LARGE")
		return
	}
	for index, cell := range headers {
		s.validateInlineNodes(cell.Children, fmt.Sprintf("%s.headers[%d].children", path, index))
	}
	for rowIndex, row := range rows {
		for cellIndex, cell := range row {
			s.validateInlineNodes(cell.Children, fmt.Sprintf("%s.rows[%d][%d].children", path, rowIndex, cellIndex))
		}
	}
}

func (s *parserState) validateExternalEmbed(block ports.BodyBlock, path string) {
	provider, _ := stringField(block, "provider")
	if provider != "image" {
		s.addError(path+".provider", "EXTERNAL_EMBED_PROVIDER_NOT_ALLOWED")
	}
	rawURL, _ := stringField(block, "url")
	safeURL, ok := sanitizeHTTPURL(rawURL)
	if !ok {
		s.addError(path+".url", "EXTERNAL_EMBED_URL_UNSAFE")
		return
	}
	s.addExternalLink(path+".url", safeURL)
}

func (s *parserState) validateAttachmentGallery(value any, path string) {
	items, ok := mapItems(value)
	if !ok {
		s.addError(path, "ATTACHMENT_GALLERY_ITEMS_INVALID")
		return
	}
	for index, item := range items {
		fileID, ok := stringField(item, "fileId")
		if !ok || strings.TrimSpace(fileID) == "" {
			s.addError(fmt.Sprintf("%s[%d].fileId", path, index), "MEDIA_REF_INVALID")
			continue
		}
		s.mediaRefs = append(s.mediaRefs, ports.MediaRef{FileID: strings.TrimSpace(fileID)})
	}
}

func (s *parserState) appendPlainText(text string) {
	if text != "" {
		s.plainTextBlocks = append(s.plainTextBlocks, text)
	}
}

func (s *parserState) addExternalLink(path string, link string) {
	if len(s.externalLinks) >= s.policy.MaxExternalLinks {
		s.addError(path, "BODY_EXTERNAL_LINK_COUNT_EXCEEDED")
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

func stringField(values map[string]any, key string) (string, bool) {
	value, ok := values[key]
	if !ok {
		return "", false
	}
	text, ok := value.(string)
	return text, ok
}

func mapItems(value any) ([]map[string]any, bool) {
	switch typed := value.(type) {
	case []map[string]any:
		return typed, true
	case []ports.BodyBlock:
		items := make([]map[string]any, 0, len(typed))
		for _, item := range typed {
			items = append(items, item)
		}
		return items, true
	case []any:
		items := make([]map[string]any, 0, len(typed))
		for _, item := range typed {
			mapped, ok := mapValue(item)
			if !ok {
				return nil, false
			}
			items = append(items, mapped)
		}
		return items, true
	default:
		return nil, false
	}
}

func mapValue(value any) (map[string]any, bool) {
	switch typed := value.(type) {
	case map[string]any:
		return typed, true
	case ports.BodyBlock:
		return typed, true
	case ports.InlineNode:
		return typed, true
	case ports.InlineMark:
		return typed, true
	default:
		return nil, false
	}
}

func inlineNodesField(values map[string]any, key string) []ports.InlineNode {
	value, ok := values[key]
	if !ok {
		return nil
	}
	if nodes, ok := value.([]ports.InlineNode); ok {
		return nodes
	}
	if rawNodes, ok := value.([]any); ok {
		nodes := make([]ports.InlineNode, 0, len(rawNodes))
		for _, rawNode := range rawNodes {
			mapped, ok := mapValue(rawNode)
			if !ok {
				return nil
			}
			nodes = append(nodes, ports.InlineNode(mapped))
		}
		return nodes
	}
	return nil
}

func marksField(values map[string]any, key string) []ports.InlineMark {
	value, ok := values[key]
	if !ok {
		return nil
	}
	if marks, ok := value.([]ports.InlineMark); ok {
		return marks
	}
	if rawMarks, ok := value.([]any); ok {
		marks := make([]ports.InlineMark, 0, len(rawMarks))
		for _, rawMark := range rawMarks {
			mapped, ok := mapValue(rawMark)
			if !ok {
				return nil
			}
			marks = append(marks, ports.InlineMark(mapped))
		}
		return marks
	}
	return nil
}

func blocksField(values map[string]any, key string) []ports.BodyBlock {
	value, ok := values[key]
	if !ok {
		return nil
	}
	if blocks, ok := value.([]ports.BodyBlock); ok {
		return blocks
	}
	if rawBlocks, ok := value.([]any); ok {
		blocks := make([]ports.BodyBlock, 0, len(rawBlocks))
		for _, rawBlock := range rawBlocks {
			mapped, ok := mapValue(rawBlock)
			if !ok {
				return nil
			}
			blocks = append(blocks, ports.BodyBlock(mapped))
		}
		return blocks
	}
	return nil
}

func tableCellsField(values map[string]any, key string) []ports.TableCell {
	value, ok := values[key]
	if !ok {
		return nil
	}
	if cells, ok := value.([]ports.TableCell); ok {
		return cells
	}
	if rawCells, ok := value.([]any); ok {
		cells := make([]ports.TableCell, 0, len(rawCells))
		for _, rawCell := range rawCells {
			cell, ok := tableCellValue(rawCell)
			if !ok {
				return nil
			}
			cells = append(cells, cell)
		}
		return cells
	}
	return nil
}

func tableRowsField(values map[string]any, key string) [][]ports.TableCell {
	value, ok := values[key]
	if !ok {
		return nil
	}
	if rows, ok := value.([][]ports.TableCell); ok {
		return rows
	}
	if rawRows, ok := value.([]any); ok {
		rows := make([][]ports.TableCell, 0, len(rawRows))
		for _, rawRow := range rawRows {
			rawCells, ok := rawRow.([]any)
			if !ok {
				return nil
			}
			row := make([]ports.TableCell, 0, len(rawCells))
			for _, rawCell := range rawCells {
				cell, ok := tableCellValue(rawCell)
				if !ok {
					return nil
				}
				row = append(row, cell)
			}
			rows = append(rows, row)
		}
		return rows
	}
	return nil
}

func tableCellValue(value any) (ports.TableCell, bool) {
	if cell, ok := value.(ports.TableCell); ok {
		return cell, true
	}
	mapped, ok := mapValue(value)
	if !ok {
		return ports.TableCell{}, false
	}
	return ports.TableCell{
		Children: inlineNodesField(mapped, "children"),
	}, true
}
