# Content Body Parser 强类型 schema 设计方案

> **给 agentic workers：** 必需子技能：实现本计划时使用 @subagent-driven-development 或 @executing-plans 逐任务推进；本计划步骤使用 checkbox 追踪。提交前必须先使用 @committing-changes。

**目标：** 将 `zhicore-content` 的 `V1BodyParser` 从动态 `map[string]any` 热路径重构为强类型正文 schema，并用现有行为测试和 benchmark 证明功能不回退、分配次数下降。

**架构：** `ports` 继续拥有 application 消费的正文解析端口和输入 DTO；`infrastructure/body` 只实现 V1 schema 的校验、规范化、外链 / 媒体提取和 hash 生成。强类型 schema 使用 `Blocks []Block` 自定义 JSON 解码 + 每种 block 独立 struct 表达子类型；parser 按 concrete subtype 做 type switch，不把所有 block 字段塞进一个宽 struct，也不增加 `BodyBlock{Value Block}` 包装层。

**技术栈：** Go 1.22、标准库 `encoding/json`、`crypto/sha256`、`testing` benchmark、`services/zhicore-content` Go module。

---

## 背景依据

- `docs/architecture/services/content/body-storage-and-publishing.md`
- `docs/architecture/services/content/adr/0002-body-blocks-no-raw-html.md`
- `docs/architecture/testing.md`
- `docs/architecture/go-service-design.md`
- `services/zhicore-content/internal/content/ports/body_parser.go`
- `services/zhicore-content/internal/content/infrastructure/body/v1_parser.go`
- `services/zhicore-content/internal/content/infrastructure/body/v1_parser_test.go`

## 当前问题

当前第一版 parser 为了快速接住 HTTP JSON 解码后的开放形态，使用了动态 alias：

```go
type BodyBlock map[string]any
type InlineNode map[string]any
type InlineMark map[string]any
```

该形态已经通过功能测试，但 benchmark 显示热路径分配偏高，尤其是：

| case | ns/op | B/op | allocs/op |
| --- | ---: | ---: | ---: |
| `many_blocks` | `1254485` | `583649` | `17766` |
| `large_table` | `835206` | `380919` | `11759` |
| `many_links` | `358136` | `176901` | `4033` |

根因不是单一函数，而是动态 map 贯穿了 JSON 输入、字段读取、子节点转换、canonical JSON 构造和错误 path 构造：

- `stringField`、`blocksField`、`inlineNodesField`、`marksField` 反复做类型断言和 `[]any` 转换。
- canonicalization 用 `map[string]any` 拼装每个块和 mark，正常路径也产生大量 map 分配。
- `fmt.Sprintf` 在正常遍历路径构造每个节点 path，即使没有错误也会分配字符串。

## 强类型 schema 目标形态

`code_block`、`image`、`paragraph`、`table` 等都是 block 的子类型。目标形态应直接表达这个事实：`Blocks` 是 JSON 边界集合，`Block` 是受控子类型接口，具体 block 由独立 struct 承载字段。

不采用宽 struct：

- 宽 struct 会把 `Code`、`FileID`、`Rows`、`Latex` 等字段集中到一个类型里，新增 block 时必须持续修改这个总类型。
- 宽 canonical struct 也会重复这个问题，新增 block 时中心结构越来越胖。
- 宽 struct 只能靠 validator 约束“哪些字段对哪个 type 有效”，类型系统没有表达子类型边界。

采用 `Blocks []Block` + subtype：

- 新增 block 时新增一个 subtype struct，并在 decoder registry、validator、canonicalizer 中登记该类型。
- 既有 block DTO 和 canonical DTO 不需要因为新 block 增加字段而变化。
- JSON decode 仍然只发生一次，不需要把正文先解成 `map[string]any`。
- 不引入 `BodyBlock{Value Block}` 这种额外包装层；集合类型 `Blocks` 自己负责从 JSON array 分派到具体 block。

```go
type PostBodyWriteInput struct {
	SchemaVersion int         `json:"schemaVersion"`
	Blocks        Blocks      `json:"blocks"`
}

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

type ListItem struct {
	Blocks Blocks `json:"blocks,omitempty"`
}

func (*ListBlock) Kind() BlockType { return BlockList }

type CodeBlock struct {
	Code     string `json:"code,omitempty"`
	Language string `json:"language,omitempty"`
}

func (*CodeBlock) Kind() BlockType { return BlockCode }

type ImageBlock struct {
	FileID string `json:"fileId,omitempty"`
	Alt    string `json:"alt,omitempty"`
}

func (*ImageBlock) Kind() BlockType { return BlockImage }

type AttachmentGalleryBlock struct {
	Items []AttachmentItem `json:"items,omitempty"`
}

func (*AttachmentGalleryBlock) Kind() BlockType { return BlockAttachmentGallery }

type InlineNode struct {
	Type  string       `json:"type"`
	Text  string       `json:"text,omitempty"`
	Marks []InlineMark `json:"marks,omitempty"`
}

type InlineMark struct {
	Type string `json:"type"`
	Href string `json:"href,omitempty"`
}

type TableCell struct {
	Children []InlineNode `json:"children"`
}

type AttachmentItem struct {
	FileID string `json:"fileId"`
}
```

上面只展示关键类型；实现时还要补齐 `TableBlock`、`TableCell`、`CollapsibleBlock`、`MathBlock`、`ExternalEmbedBlock` 和 `UnsupportedBlock`。

`Blocks.UnmarshalJSON` 用 `json.RawMessage` 保留每个 block 的原始 JSON，逐个读取轻量 discriminator，再按 registry 解到具体 struct：

```go
func (bs *Blocks) UnmarshalJSON(data []byte) error {
	var raws []json.RawMessage
	if err := json.Unmarshal(data, &raws); err != nil {
		return err
	}

	out := make([]Block, 0, len(raws))
	for _, raw := range raws {
		block, err := decodeBlock(raw)
		if err != nil {
			return err
		}
		out = append(out, block)
	}

	*bs = out
	return nil
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
```

`UnsupportedBlock` 的目的不是接受未知类型，而是把 `BLOCK_TYPE_NOT_ENABLED` 留给 parser 生成统一的字段级校验错误；它不能进入 canonical JSON。

因为具体 subtype struct 不保存 `type` 字段，`Blocks.MarshalJSON` 如果实现，必须在输出时注入 `type`：

```go
func (bs Blocks) MarshalJSON() ([]byte, error) {
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
```

`Blocks.MarshalJSON` 只用于测试 fixture 和 canonical 前的调试输出，正文 hash 不直接复用输入 marshal。canonical JSON 仍由 parser 明确构造，保证 unknown fields、字段顺序、URL 规范化和 trim 规则由 Content 控制。

## 文件结构

- 修改：`services/zhicore-content/internal/content/ports/body_parser.go`
  - 删除 `BodyBlock map[string]any` alias，改为 `type Blocks []Block`。
  - 新增 `Block` 接口、`BlockType` 常量和每种 block 的 subtype struct。
  - 为 `Blocks` 新增 `UnmarshalJSON` / `MarshalJSON`，让 HTTP JSON 可以直接解到 typed DTO。
- 修改：`services/zhicore-content/internal/content/infrastructure/body/v1_parser.go`
  - 删除动态 map 字段读取 helper。
  - 使用 subtype type switch 直接遍历、校验和规范化。
  - 将 canonicalization 改为 subtype-specific canonical struct，不使用中心化宽 canonical struct。
  - 正常路径避免构造完整 path 字符串，只在错误路径或可能报错的分支构造。
- 修改：`services/zhicore-content/internal/content/infrastructure/body/v1_parser_test.go`
  - 将 fixture builder 改为强类型构造。
  - 保留 JSON decode 测试，证明 `encoding/json` 能直接解到 typed DTO。
  - 保留 benchmark case，并记录重构前后指标。
- 可选新增：`services/zhicore-content/internal/content/infrastructure/body/v1_schema.go`
  - 不建议。schema DTO 应留在 `ports`，否则 application-facing input 会反向依赖 infrastructure。
- 可选新增：`services/zhicore-content/internal/content/ports/body_blocks.go`
  - 当 `ports/body_parser.go` 变长时，把 block subtype DTO 拆到同包新文件。

## 任务 1：锁定现有 parser 行为和 baseline

**测试立场：** TDD / characterization - parser 是安全敏感输入校验器，属于 R3。

- [ ] **步骤 1：运行当前行为测试**

  运行：`cd services/zhicore-content && go test ./internal/content/infrastructure/body`

  预期：通过，证明重构前行为基线可用。

- [ ] **步骤 2：运行当前 benchmark 并保存输出**

  运行：`cd services/zhicore-content && go test -run Test -bench=BenchmarkV1BodyParser -benchmem ./internal/content/infrastructure/body`

  预期：输出包含 `ns/op`、`B/op`、`allocs/op`；把结果临时记录在任务说明或 review 证据中。

- [ ] **步骤 3：确认测试文件规模**

  运行：`python3 scripts/check-test-size.py --files services/zhicore-content/internal/content/infrastructure/body/v1_parser_test.go`

  预期：通过；如果接近 400 行，后续任务优先拆 benchmark helper 到独立测试文件。

## 任务 2：把端口 DTO 改为 block subtype schema

**测试立场：** TDD - 这是 application-facing contract 的类型重构，先让现有测试暴露编译失败，再按目标类型修复。

- [ ] **步骤 1：修改 `ports` 中的类型定义**

  把动态 alias 替换为 `Blocks []Block`、`Block` 接口、`BlockType` 常量和每个 block subtype struct，保留 `PostBodyWriteInput`、`BodyValidationPolicy`、`NormalizedBody`、`BodyValidationError` 的公开语义。

- [ ] **步骤 2：运行 parser 包测试，确认进入预期红灯**

  运行：`cd services/zhicore-content && go test ./internal/content/infrastructure/body`

  预期：编译失败，失败点来自测试 fixture 和 parser 实现仍在按 map 访问字段。

- [ ] **步骤 3：先修测试 fixture 到 subtype 构造**

  示例：

  ```go
  Blocks{
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
  }
  ```

- [ ] **步骤 4：保留 JSON decode 测试**

  `TestV1BodyParserNormalizesDecodedJSONBody` 必须继续从 raw JSON 直接 `json.Unmarshal` 到 `PostBodyWriteInput`，证明 handler 层不需要先解成 `map[string]any`。

## 任务 3：重写 typed traversal 和 validation

**测试立场：** TDD - 现有测试已经红灯，按现有行为逐步变绿。

- [ ] **步骤 1：删除 map helper 的生产调用**

  删除或停止使用 `stringField`、`mapItems`、`mapValue`、`inlineNodesField`、`marksField`、`blocksField`、`tableCellsField`、`tableRowsField`、`tableCellValue`。

- [ ] **步骤 2：把 block 校验改为 subtype type switch**

  `validateBlock` 接收 `ports.Block`，按 `block.(type)` 分支：

  - `*ports.ParagraphBlock` / `*ports.HeadingBlock` 读取 `Children`。
  - `*ports.QuoteBlock` / `*ports.CollapsibleBlock` 读取 `Blocks`，并保留 `MaxContainerDepth`。
  - `*ports.ListBlock` 读取 `Items[index].Blocks`。
  - `*ports.TableBlock` 读取 `Headers` 和 `Rows`。
  - `*ports.ImageBlock` 读取 `FileID`。
  - `*ports.ExternalEmbedBlock` 读取 `Provider` 和 `URL`。
  - `*ports.AttachmentGalleryBlock` 读取 `Items`。
  - `*ports.UnsupportedBlock` 返回 `BLOCK_TYPE_NOT_ENABLED`。

- [ ] **步骤 3：把 inline 和 mark 校验改为 typed field 访问**

  `validateInlineNodes` 读取 `node.Type`、`node.Text`、`node.Marks`；`validateMarks` 读取 `mark.Type`、`mark.Href`。

- [ ] **步骤 4：运行 parser 包测试**

  运行：`cd services/zhicore-content && go test ./internal/content/infrastructure/body`

  预期：除 canonical JSON 形态相关断言外，其余安全校验、limit、JSON decode 和错误截断行为通过。

## 任务 4：重写 subtype canonicalization

**测试立场：** TDD - `contentHash`、`CanonicalJSON` 和 unknown field 丢弃属于正文一致性 contract。

- [ ] **步骤 1：引入每种 block 的 canonical struct**

  使用内部非导出类型表达 canonical 输出，不再用 `map[string]any`，也不使用中心化宽 `canonicalBlock`：

  ```go
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

  type canonicalImageBlock struct {
    Type   ports.BlockType `json:"type"`
    FileID string          `json:"fileId"`
    Alt    string          `json:"alt,omitempty"`
  }
  ```

  每个 subtype 只拥有自己的 canonical 字段。新增 block 时新增一个 `canonicalXBlock` 和一个 canonicalizer 分支，不修改既有 canonical struct。

- [ ] **步骤 2：保留 URL 规范化和文件 ID trim**

  `link.href`、`external_embed.url` 继续用 `sanitizeHTTPURL` 输出规范化 URL；`image.fileId`、`attachment_gallery.items[].fileId` 继续 `strings.TrimSpace`。

- [ ] **步骤 3：运行 canonical 行为测试**

  运行：`cd services/zhicore-content && go test ./internal/content/infrastructure/body -run 'TestV1BodyParserCanonicalJSON|TestV1BodyParserNormalizes'`

  预期：通过；如果 canonical byte 序因为 struct 字段顺序发生变化，只要 unknown fields 被丢弃、hash 稳定且 schema 文档接受，就更新测试为行为断言，不要为了旧 map 顺序保留动态 map。

## 任务 5：降低正常路径分配

**测试立场：** R3 性能重构 - 不新增脆弱的硬阈值单元测试，用 benchmark 对比和 review 证据证明。

- [ ] **步骤 1：预分配输出切片**

  在 `parserState` 初始化时按输入规模保守预分配：

  - `plainTextBlocks` 容量不超过 `min(len(input.Blocks), policy.MaxBlocks)`。
  - `mediaRefs` 和 `externalLinks` 使用小容量初值，例如 0 或 4；不要按最大阈值一次性分配。
  - canonical block slice 按当前层 `len(blocks)` 分配。

- [ ] **步骤 2：只在错误路径构造详细 path**

  正常遍历用数字 index 传递上下文；只有需要 `addError` 时再构造 path。短期可先保留少量 `fmt.Sprintf`，但 `many_blocks` 和 `large_table` 的内层循环不应无条件格式化 path。

- [ ] **步骤 3：删除正常路径 map 分配**

  canonicalization 和 traversal 不应再出现 `map[string]any{}`。允许 `UnmarshalJSON` 使用轻量 discriminator struct 和 `json.RawMessage`；测试 helper 也应迁移到 typed builder，避免 benchmark 输入本身污染分配观察。

- [ ] **步骤 4：运行 benchmark 对比**

  运行：`cd services/zhicore-content && go test -run Test -bench=BenchmarkV1BodyParser -benchmem ./internal/content/infrastructure/body`

  预期：

  - `small`、`medium`、`near_limit`、`many_blocks`、`large_table`、`many_links` 均保持毫秒级以内或不明显变慢。
  - `many_blocks` 和 `large_table` 的 `allocs/op` 相比 baseline 明显下降。
  - 如果 `ns/op` 降低但 `allocs/op` 不降，继续检查 canonicalization 是否仍在构造 map 或 path 字符串。

## 任务 6：收口验证和文档同步

**测试立场：** 文档 + 重构验证，R1/R3 混合。

- [ ] **步骤 1：运行 parser 包测试**

  运行：`cd services/zhicore-content && go test ./internal/content/infrastructure/body`

  预期：通过。

- [ ] **步骤 2：运行 content 服务全量测试**

  运行：`cd services/zhicore-content && go test ./...`

  预期：通过。

- [ ] **步骤 3：运行测试规模检查**

  运行：`python3 scripts/check-test-size.py --files services/zhicore-content/internal/content/infrastructure/body/v1_parser_test.go`

  预期：通过；如果失败，按行为拆分 `v1_parser_test.go` 和 `v1_parser_benchmark_test.go`。

- [ ] **步骤 4：运行结构检查**

  运行：`bash scripts/check-structure.sh`

  预期：`structure ok`。

- [ ] **步骤 5：更新长期事实源**

  如果实现后确认强类型 schema 成为正式事实，在 `docs/architecture/services/content/body-storage-and-publishing.md` 的 `BodyParserRegistry` 或 blocks schema 段落补一句：V1 parser 在 Go 端以 typed DTO 解码，`map[string]any` 不进入 parser 热路径。

## 架构适配评估

- 端口边界明确：application 仍只依赖 `ports.BodyParserRegistry` 和 `ports.PostBodyWriteInput`，不导入 `infrastructure/body`。
- schema owner 明确：V1 正文 schema 属于 Content 服务内 ports DTO，不提升到 `libs/contracts`，因为它不是跨服务同步调用 contract。
- 安全边界明确：raw HTML / unknown fields 不进入 canonical JSON；URL scheme、provider 白名单、media ref 校验仍由 parser 统一负责。
- 性能目标明确：本计划不是只让测试变绿，还必须用 `benchmem` 对比 `B/op` 和 `allocs/op`，尤其关注 `many_blocks`、`large_table` 和 `many_links`。
- 结构收敛明确：如果执行后仍保留 `BodyBlock map[string]any`、`BodyBlock{Value Block}` 包装层、宽 block 字段集合、宽 `canonicalBlock` 字段集合或 canonicalization 的 `map[string]any`，则本计划未完成。

## 风险和取舍

- subtype 方案比宽 struct 多一个 `Blocks.UnmarshalJSON` decoder registry，但它能把 block 子类型边界放进类型系统，避免一个 DTO 随 block 类型增长而膨胀。
- 新增 block 仍然必须修改 decoder registry、validator 和 canonicalizer。原因是安全校验和 canonical JSON 是显式白名单，不能让新 block 自动绕过校验进入存储。
- `encoding/json` 默认忽略未知字段，这与当前 canonical unknown fields drop 行为一致；如果未来要“未知字段直接报错”，应作为独立 contract 变更处理。
- canonical JSON byte 顺序可能因从 map 切到 struct 发生变化，从而改变 `contentHash`。当前正文 parser 尚未接入生产写路径，可以接受；接入后再改 hash 规则必须走 migration / compatibility 设计。
- benchmark 不应写死机器相关耗时阈值；本轮只用相对 baseline 和 `allocs/op` / `B/op` 变化做判断。
