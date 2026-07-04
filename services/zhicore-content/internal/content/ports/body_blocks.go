package ports

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
)

type Blocks []Block

type Block interface {
	Kind() BlockType
}

type BlockType string

const (
	BlockParagraph         BlockType = "paragraph"
	BlockHeading           BlockType = "heading"
	BlockQuote             BlockType = "quote"
	BlockList              BlockType = "list"
	BlockCode              BlockType = "code_block"
	BlockTable             BlockType = "table"
	BlockCollapsible       BlockType = "collapsible"
	BlockMath              BlockType = "math"
	BlockImage             BlockType = "image"
	BlockExternalEmbed     BlockType = "external_embed"
	BlockAttachmentGallery BlockType = "attachment_gallery"
)

type ParagraphBlock struct {
	Children []InlineNode `json:"children,omitempty"`
}

func (*ParagraphBlock) Kind() BlockType { return BlockParagraph }

type HeadingBlock struct {
	Level    int          `json:"level,omitempty"`
	Children []InlineNode `json:"children,omitempty"`
}

func (*HeadingBlock) Kind() BlockType { return BlockHeading }

type QuoteBlock struct {
	Blocks Blocks `json:"blocks,omitempty"`
}

func (*QuoteBlock) Kind() BlockType { return BlockQuote }

type ListBlock struct {
	Items []ListItem `json:"items,omitempty"`
}

func (*ListBlock) Kind() BlockType { return BlockList }

type ListItem struct {
	Blocks Blocks `json:"blocks,omitempty"`
}

type CodeBlock struct {
	Code     string `json:"code,omitempty"`
	Language string `json:"language,omitempty"`
}

func (*CodeBlock) Kind() BlockType { return BlockCode }

type TableBlock struct {
	Headers []TableCell   `json:"headers,omitempty"`
	Rows    [][]TableCell `json:"rows,omitempty"`
}

func (*TableBlock) Kind() BlockType { return BlockTable }

type TableCell struct {
	Children []InlineNode `json:"children"`
}

type CollapsibleBlock struct {
	Title  []InlineNode `json:"title,omitempty"`
	Blocks Blocks       `json:"blocks,omitempty"`
}

func (*CollapsibleBlock) Kind() BlockType { return BlockCollapsible }

type MathBlock struct {
	Latex string `json:"latex,omitempty"`
}

func (*MathBlock) Kind() BlockType { return BlockMath }

type ImageBlock struct {
	FileID string `json:"fileId,omitempty"`
	Alt    string `json:"alt,omitempty"`
}

func (*ImageBlock) Kind() BlockType { return BlockImage }

type ExternalEmbedBlock struct {
	Provider string `json:"provider,omitempty"`
	URL      string `json:"url,omitempty"`
	Title    string `json:"title,omitempty"`
}

func (*ExternalEmbedBlock) Kind() BlockType { return BlockExternalEmbed }

type AttachmentGalleryBlock struct {
	Items []AttachmentItem `json:"items,omitempty"`
}

func (*AttachmentGalleryBlock) Kind() BlockType { return BlockAttachmentGallery }

type AttachmentItem struct {
	FileID string `json:"fileId"`
}

type UnsupportedBlock struct {
	Type BlockType
}

func (b *UnsupportedBlock) Kind() BlockType { return b.Type }

type InlineNode struct {
	Type  string       `json:"type"`
	Text  string       `json:"text,omitempty"`
	Marks []InlineMark `json:"marks,omitempty"`
}

type InlineMark struct {
	Type string `json:"type"`
	Href string `json:"href,omitempty"`
}

func (bs *Blocks) UnmarshalJSON(data []byte) error {
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		*bs = nil
		return nil
	}

	var raws []json.RawMessage
	if err := json.Unmarshal(data, &raws); err != nil {
		return err
	}

	blocks := make([]Block, 0, len(raws))
	for _, raw := range raws {
		block, err := decodeBlock(raw)
		if err != nil {
			return err
		}
		blocks = append(blocks, block)
	}

	*bs = blocks
	return nil
}

func (bs Blocks) MarshalJSON() ([]byte, error) {
	if bs == nil {
		return []byte("null"), nil
	}

	raws := make([]json.RawMessage, 0, len(bs))
	for _, block := range bs {
		raw, err := marshalBlock(block)
		if err != nil {
			return nil, err
		}
		raws = append(raws, raw)
	}
	return json.Marshal(raws)
}

func decodeBlock(raw json.RawMessage) (Block, error) {
	var head struct {
		Type BlockType `json:"type"`
	}
	if err := json.Unmarshal(raw, &head); err != nil {
		return nil, err
	}
	decode, ok := blockDecoders[head.Type]
	if !ok {
		return &UnsupportedBlock{Type: head.Type}, nil
	}
	return decode(raw)
}

var blockDecoders = map[BlockType]func(json.RawMessage) (Block, error){
	BlockParagraph:         decodeTypedBlock[*ParagraphBlock],
	BlockHeading:           decodeTypedBlock[*HeadingBlock],
	BlockQuote:             decodeTypedBlock[*QuoteBlock],
	BlockList:              decodeTypedBlock[*ListBlock],
	BlockCode:              decodeTypedBlock[*CodeBlock],
	BlockTable:             decodeTypedBlock[*TableBlock],
	BlockCollapsible:       decodeTypedBlock[*CollapsibleBlock],
	BlockMath:              decodeTypedBlock[*MathBlock],
	BlockImage:             decodeTypedBlock[*ImageBlock],
	BlockExternalEmbed:     decodeTypedBlock[*ExternalEmbedBlock],
	BlockAttachmentGallery: decodeTypedBlock[*AttachmentGalleryBlock],
}

func decodeTypedBlock[T Block](raw json.RawMessage) (Block, error) {
	var block T
	if err := json.Unmarshal(raw, &block); err != nil {
		return nil, err
	}
	return block, nil
}

func marshalBlock(block Block) ([]byte, error) {
	if isNilBlock(block) {
		return nil, fmt.Errorf("nil body block")
	}

	switch typed := block.(type) {
	case *ParagraphBlock:
		return json.Marshal(struct {
			Type BlockType `json:"type"`
			*ParagraphBlock
		}{Type: BlockParagraph, ParagraphBlock: typed})
	case *HeadingBlock:
		return json.Marshal(struct {
			Type BlockType `json:"type"`
			*HeadingBlock
		}{Type: BlockHeading, HeadingBlock: typed})
	case *QuoteBlock:
		return json.Marshal(struct {
			Type BlockType `json:"type"`
			*QuoteBlock
		}{Type: BlockQuote, QuoteBlock: typed})
	case *ListBlock:
		return json.Marshal(struct {
			Type BlockType `json:"type"`
			*ListBlock
		}{Type: BlockList, ListBlock: typed})
	case *CodeBlock:
		return json.Marshal(struct {
			Type BlockType `json:"type"`
			*CodeBlock
		}{Type: BlockCode, CodeBlock: typed})
	case *TableBlock:
		return json.Marshal(struct {
			Type BlockType `json:"type"`
			*TableBlock
		}{Type: BlockTable, TableBlock: typed})
	case *CollapsibleBlock:
		return json.Marshal(struct {
			Type BlockType `json:"type"`
			*CollapsibleBlock
		}{Type: BlockCollapsible, CollapsibleBlock: typed})
	case *MathBlock:
		return json.Marshal(struct {
			Type BlockType `json:"type"`
			*MathBlock
		}{Type: BlockMath, MathBlock: typed})
	case *ImageBlock:
		return json.Marshal(struct {
			Type BlockType `json:"type"`
			*ImageBlock
		}{Type: BlockImage, ImageBlock: typed})
	case *ExternalEmbedBlock:
		return json.Marshal(struct {
			Type BlockType `json:"type"`
			*ExternalEmbedBlock
		}{Type: BlockExternalEmbed, ExternalEmbedBlock: typed})
	case *AttachmentGalleryBlock:
		return json.Marshal(struct {
			Type BlockType `json:"type"`
			*AttachmentGalleryBlock
		}{Type: BlockAttachmentGallery, AttachmentGalleryBlock: typed})
	case *UnsupportedBlock:
		return json.Marshal(struct {
			Type BlockType `json:"type"`
		}{Type: typed.Type})
	default:
		return nil, fmt.Errorf("unsupported body block %T", block)
	}
}

func isNilBlock(block Block) bool {
	if block == nil {
		return true
	}
	value := reflect.ValueOf(block)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}
