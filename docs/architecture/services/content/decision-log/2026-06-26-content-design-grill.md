# Content 设计压测问答重建日志

本文按 2026-06-26 对 Content 正文、草稿、发布、blocks schema、错误契约和链接预览的 84 轮压测讨论重建。它不是逐字 transcript，而是从当前对话、最终设计文档和 ADR 反推出的“问题 -> 结论 -> 原因 / 修正”记录，用于后续复盘。

相关事实源：

- [Content CONTEXT](../CONTEXT.md)
- [正文存储与发布流程](../body-storage-and-publishing.md)
- [数据、事件和契约设计](../data-events-contracts.md)
- [ADR 0001](../adr/0001-body-pointer-publish-atomicity.md)
- [ADR 0002](../adr/0002-body-blocks-no-raw-html.md)
- [ADR 0003](../adr/0003-link-preview-deferred.md)

## 重建问题清单

1. **发布后的正文是否必须是不可变快照？**
   结论：必须。发布读取 `published_body_id` 指向的 immutable snapshot。
   原因：避免编辑中草稿污染线上正文。

2. **已发布文章编辑时直接覆盖线上正文，还是创建编辑草稿？**
   结论：创建编辑草稿，二次发布后再替换线上正文。
   原因：编辑过程、保存失败或审核失败不能影响读者看到的内容。

3. **发布正文是否保留完整历史版本？**
   结论：普通文章不保留完整历史版本；该问题后来被第 4、5、11、12 问细化。
   原因：普通文章长期版本库成本高，价值主要是短期排障和恢复。

4. **共建类文档是否需要保留历史版本？**
   结论：共建文档保留历史版本；普通个人文章不保留。
   原因：共建文档需要 diff、回滚、作者 attribution 和变更说明。

5. **共建文档是独立内容类型还是普通文章开关？**
   结论：作为独立内容类型或聚合预留。
   原因：协作权限、revision history、回滚和治理规则与普通文章不同。

6. **共建文档是否需要完整 Git 分支模型？**
   结论：不需要；参考论坛 Wiki / 共建帖，采用线性 revision history。
   原因：branch / merge / rebase 会显著增加编辑和冲突处理复杂度。

7. **共建文档编辑权限怎么来？**
   结论：目标设计预留角色、信任等级和邀请协作者；第一阶段不一定实现。
   原因：这属于社区治理能力，不应阻塞普通文章发布链路。

8. **普通文章和共建文档是否共用主内容模型？**
   结论：共用 `posts` 主模型，通过 `content_type` / `version_policy` 扩展；共建 revision 另建扩展。
   原因：复用文章生命周期和列表能力，同时避免普通文章背负 revision 复杂度。

9. **发布时 Mongo 固化正文和 PG 切换指针的顺序？**
   结论：先写 Mongo snapshot，再用 PG transaction 切 `published_body_id`。
   原因：Mongo 失败时 PG 不变；PG 失败时新 snapshot 是 orphan，可清理。

10. **PG 切换失败后的 orphan body 是否删除？会不会删 draft？**
    结论：删除未被 PG 引用的新 snapshot / candidate，不删除当前 draft。
    原因：用户保存的 draft 仍是编辑真相，orphan 只是失败发布副本。

11. **PG 成功后如何确定 Mongo 中哪个 body 是真实线上正文？**
    结论：只看 PG `published_body_id`，不看 Mongo `role=published`。
    原因：PG 是可见性真相源，避免多个 Mongo snapshot 产生歧义。

12. **个人文章保留那么多历史版本做什么？**
    结论：普通个人文章不保留完整历史版本。
    原因：除排障和短期恢复外价值有限，存储和治理成本更高。

13. **个人文章是否需要正文版本号？**
    结论：不需要产品意义上的版本号；使用内部 `body_id` / UUID 引用。
    原因：body_id 用于原子切换正文指针，不向用户承诺历史版本能力。

14. **发布成功后 draft 立即删除还是保留？**
    结论：发布成功后清空 `draft_*`，旧 draft 进入清理；下次编辑再派生新 draft。
    原因：发布后 draft 与线上内容一致，继续保留会制造状态歧义和空间占用。

15. **是否允许只创建 PG 草稿但没有 Mongo 正文？**
    结论：允许空草稿占位，但不能发布，并可 TTL 清理。
    原因：用户打开编辑器但未输入内容时，不应立即制造 Mongo body。

16. **草稿 quota 按数量还是大小限制？**
    结论：数量、单篇大小和总大小都限制。
    原因：防止大量小草稿和单个大草稿分别拖垮列表、存储和清理。

17. **Mongo snapshot 写成功但 PG 切换失败，如何清理？**
    结论：发布失败，snapshot 异步清理；可 best-effort 删除但不阻塞。
    原因：清理失败不应变成用户可见的第二个错误。

18. **发布后马上再次编辑，旧 draft 清理会不会误删新 draft？**
    结论：所有清理必须按 `body_id` 精确删除，并先查 PG 当前引用。
    原因：按 `post_id + role=draft` 删除会误删新编辑周期的 draft。

19. **`draft_body_ref` 是否需要一个 ID？**
    结论：需要，统一命名为 `draft_body_id`，使用 UUID。
    原因：PG 和 Mongo 是两个存储，PG 只能持有稳定正文引用。

20. **draft 生命周期内 UUID 是复用还是每次保存新建？**
    结论：最初倾向复用，后来第 27 问修正为每次保存 copy-on-write 新 UUID。
    原因：原地覆盖会在 Mongo 成功、PG 失败时破坏旧草稿。

21. **已发布文章再次编辑时是否新建 draft UUID？**
    结论：新建 draft UUID。
    原因：上一轮发布后的旧 draft 已进入清理流程，不能复用。

22. **重新发布后旧 published snapshot 是否清理？**
    结论：异步清理旧 published snapshot。
    原因：PG 已切到新正文，旧 snapshot 不再是线上事实，只需防误删即可。

23. **PG 是否保存 `published_body_hash` / `draft_body_hash`？**
    结论：两者都保存。
    原因：用于一致性检查、幂等辅助、冲突检测和排查。

24. **`content_hash` 用途是什么？**
    结论：一致性指纹和幂等辅助，不是安全证明。
    原因：hash 可判断内容是否变化，但不能替代权限、审核或防篡改机制。

25. **同一篇文章是否允许多设备编辑同一个 draft？**
    结论：允许打开，但保存时用 `base_post_version` / `base_draft_hash` 乐观冲突检测。
    原因：防止一个设备静默覆盖另一个设备的保存结果。

26. **发布时是否也校验 draft id/hash？**
    结论：必须校验 `draft_body_id` 和 `draft_body_hash`。
    原因：发布者必须发布自己确认过的草稿状态。

27. **发布前是否校验 Mongo draft hash 与 PG draft hash 一致？**
    结论：必须校验。
    原因：PG 指针和 Mongo body 不一致时属于数据事故，应阻止发布并 repair。

28. **保存草稿时 Mongo 写成功但 PG 更新失败怎么办？**
    结论：改为草稿 copy-on-write，每次保存先写新 Mongo draft，再 PG 切指针。
    原因：避免原地覆盖导致 PG 旧 hash 对不上 Mongo 新内容。

29. **copy-on-write 草稿是否需要 autosave 节流？**
    结论：需要前端 debounce、后端节流和 hash no-op。
    原因：每次保存都新建 body，必须控制写入和清理压力。

30. **保存草稿成功时是否写 cleanup task？**
    结论：PG transaction 同事务写旧 draft cleanup task。
    原因：避免 PG 已不引用旧 draft 但清理任务丢失。

31. **正文清理任务复用 `domain_event_task` 还是单独表？**
    结论：单独 `content_body_cleanup_tasks`。
    原因：资源回收与业务投影任务语义不同，需要 body_id、reason、retry 等字段。

32. **内容修复任务和清理任务是否分开？**
    结论：分开，使用 `content_body_repair_tasks`。
    原因：repair 是数据一致性事故，cleanup 是资源回收。

33. **草稿列表和文章列表是否读取 Mongo 正文？**
    结论：不读 Mongo，只读 PG 快照字段。
    原因：列表批量读 Mongo 成本高且一致性差。

34. **标题、摘要、封面属于 PG 还是 Mongo？**
    结论：PG 是真相源，Mongo 可冗余但不作为列表事实。
    原因：这些字段用于列表、搜索事件、通知和分享卡片。

35. **发布失败时元数据是否也要原子回滚？**
    结论：需要 `published_*` / `draft_*` 元数据分离，发布由 PG 事务原子切换。
    原因：用户视角发布要么全部上线，要么线上完全不变。

36. **首次草稿只写 `draft_*` 吗？**
    结论：是，首次发布成功后才写 `published_*`。
    原因：未发布草稿没有线上可见事实。

37. **已发布文章打开编辑器时是否立即创建 draft？**
    结论：不创建；首次保存才创建服务端 draft，前端可本地临时保存。
    原因：用户只是打开编辑器不应占用 quota 或制造清理任务。

38. **本地草稿和服务端草稿冲突时谁为准？**
    结论：服务端为准，前端提示用户恢复本地或丢弃；后端仍用版本/hash 判冲突。
    原因：本地草稿只是 UX，不是服务端事实。

39. **封面文件生命周期如何处理？**
    结论：Content 不直接物理删除 Upload 文件，只释放引用 / 发事件，Upload 清理。
    原因：Upload 拥有文件对象和物理清理语义。

40. **发布前是否校验封面文件 facts？**
    结论：封面非必填；上传在前端点击上传时完成，保存草稿时绑定，发布只做引用防线。
    原因：发布链路不应承担上传流程。

41. **上传封面但草稿未保存，谁清理 orphan 文件？**
    结论：Upload 用 temporary / unbound TTL 清理。
    原因：Content 还没记录引用，无法负责未绑定文件。

42. **保存草稿时如何确认 Upload 引用？**
    结论：同步校验 facts，PG commit 后异步确认引用。
    原因：不能把 Upload 调用放进 Content PG transaction。

43. **正文和元数据是否必须一起保存？**
    结论：允许分开保存；可提供组合 SaveDraft。
    原因：标题/封面变化不应强制重写 Mongo 正文。

44. **`post_version` 控制元数据还是整篇文章？**
    结论：控制整篇 `posts` 行。
    原因：任何 PG 状态变更都要参与乐观锁。

45. **A 改标题、B 改正文是否允许字段级合并？**
    结论：第一阶段不自动合并，保留后续优化。
    原因：字段级合并会带来复杂草稿状态语义。

46. **发布时是否校验 `base_post_version`？**
    结论：必须。
    原因：发布确认页看到的草稿必须和提交时的草稿一致。

47. **内容审核同步还是异步？**
    结论：第一阶段发布前同步基础审核，异步深度审核后续扩展。
    原因：明显违规内容不应先上线再下架。

48. **审核通过后到 PG commit 之间正文是否可能被改？**
    结论：PG transaction 提交前再次校验 `post_version / draft_body_id / draft_hash`。
    原因：审核期间其他设备可能保存新草稿。

49. **发布事件是否携带正文全文？**
    结论：不携带全文，只带 body id/hash 和轻量元数据。
    原因：事件持久化和重试会放大正文全文的存储、隐私和治理成本。

50. **Search 拉正文不可用时 ack 还是重试？**
    结论：重试 / DLQ，不当作成功。
    原因：发布事件表示 Search 索引应最终收敛。

51. **Ranking / Notification 是否读取正文？**
    结论：不读取正文，只消费轻量字段。
    原因：它们只需要发布事实、作者、标题、摘要、封面等。

52. **Search 内部接口是否暴露草稿？**
    结论：不暴露，只暴露当前 published body。
    原因：Search 只能索引线上可见正文。

53. **`status=published` 是否允许 `published_body_id` 为空？**
    结论：不允许。
    原因：published 文章必须有可读正文指针和 hash。

54. **`status=draft` 是否允许没有 `draft_body_id`？**
    结论：允许，表示空草稿占位；不可发布。
    原因：用户刚创建草稿时可能还没有正文。

55. **summary 是用户填还是系统生成？**
    结论：用户优先；AI summary 可配置、用户触发并接受后才写入；默认非必填。
    原因：AI 不应隐式覆盖用户摘要，也不应成为发布硬依赖。

56. **title 是否必填？**
    结论：发布时必填，草稿阶段可为空。
    原因：列表、详情、搜索、通知和分享卡片需要稳定标题。

57. **正文最小要求是什么？**
    结论：普通文章发布时有效 `plain_text` 至少 10 个 rune；媒体不能替代文字正文。
    原因：媒体可作为内嵌内容，但普通文章不能只有图片/视频没有文字说明。

58. **10 个字如何计数？**
    结论：按去掉空白和格式标记后的有效文本 rune 数。
    原因：第一阶段简单稳定，后续再做 CJK / 英文词权重。

59. **媒体引用保存时校验还是发布时校验？**
    结论：internal media 保存/发布校验 Upload facts；external media 只做格式和安全校验。
    原因：外部媒体可用性不归 Content 控制。

60. **是否允许外部媒体 URL？**
    结论：允许，但 `external_embed` 使用 provider 白名单；普通外链安全渲染。
    原因：不允许任意 iframe / HTML，避免 XSS 和嵌入风险。

61. **外部媒体 / 链接是否需要 SSRF 防护？**
    结论：任何后端抓取都必须走 SSRF-safe fetcher。
    原因：用户提交 URL 可能指向内网、metadata endpoint 或恶意重定向。

62. **链接预览是什么、由谁做？**
    结论：第一阶段不做；后续由后端异步生成，前端只消费缓存 preview。
    原因：后端预览涉及 SSRF、缓存、重试和治理，前端抓取不稳定且不可控。

63. **链接预览是否需要记录后续项？**
    结论：需要，已记录 todo 和 ADR 0003。
    原因：这是明确的后续能力，不应在当前实现中误做。

64. **正文格式用 Markdown、HTML 还是 blocks？**
    结论：结构化 blocks；可支持 Markdown 输入但存储为 blocks。
    原因：blocks 更适合媒体、审核、搜索、AI summary 和 schema migration。

65. **`plainText` 保存还是动态计算？**
    结论：保存时计算并持久化到 MongoDB，PG 只存长度等摘要字段。
    原因：发布校验、搜索、审核等多个路径都需要一致的 plain_text。

66. **`content_hash` 对什么计算？**
    结论：对 canonical blocks 计算 SHA-256。
    原因：避免字段顺序、空字段、前端临时字段导致 hash 漂移。

67. **第一阶段支持哪些 block 类型？**
    结论：开放基础展示块、媒体块、table、collapsible、math、attachment_gallery；mention / poll / custom_widget 只预留。
    原因：互动型块会牵涉通知、投票状态、权限和治理。

68. **`math` 用什么格式？**
    结论：LaTeX 字符串。
    原因：简单、成熟，前端用 KaTeX / MathJax 渲染，后端不执行公式。

69. **`table` 支持复杂合并单元格吗？**
    结论：第一阶段只支持简单二维表。
    原因：rowspan / colspan 会显著增加编辑器和渲染复杂度。

70. **`collapsible` 是什么，是否做？**
    结论：它是可展开/收起内容块，不是拖拽；第一阶段开放，最大嵌套深度 2。
    原因：它是展示能力，可控；拖拽排序属于前端交互。

71. **拖拽排序是否进入后端模型？**
    结论：不进入，只保存最终 blocks 数组顺序。
    原因：拖拽是编辑器交互，后端只关心最终内容结构。

72. **`attachment_gallery` 是否允许外部附件 URL？**
    结论：只允许 Upload `file_id`。
    原因：附件涉及下载、安全、文件名、大小、类型、权限和清理。

73. **`external_embed` 允许哪些 provider？**
    结论：使用配置白名单，不允许任意 iframe。
    原因：前端根据 provider + URL 生成安全 embed，而不是信任用户 HTML。

74. **`code_block.language` 是否白名单？**
    结论：不强白名单，只做格式和长度限制。
    原因：语言只是高亮 hint，不是执行逻辑。

75. **marks 支持哪些？**
    结论：`bold`、`italic`、`underline`、`strike`、`inline_code`、`link`。
    原因：覆盖基础富文本需求，同时禁止任意 style。

76. **是否允许 raw HTML？**
    结论：不允许。
    原因：raw HTML 会扩大 XSS、style 污染、script / iframe 注入风险。

77. **blocks 是否存 `schemaVersion`？**
    结论：必须存。
    原因：未来 block 字段和渲染规则会变，需要可迁移。

78. **schemaVersion 升级读兼容还是后台迁移？**
    结论：两者都要；先读兼容，后台分批迁移。
    原因：避免一次性迁移阻塞上线，也保留最终收敛能力。

79. **schema migration 是否原地改 Mongo body？**
    结论：当前 published / draft body 不原地改，仍走 copy-on-write。
    原因：原地改坏会直接损坏线上正文。

80. **copy-on-write migration 会不会占用过高空间？**
    结论：会短期放大；通过分批、并发限制、空间水位、优先活跃内容控制。
    原因：空间风险比线上正文被原地写坏更可控。

81. **正文解析是否按版本用设计模式实现？**
    结论：使用 Strategy + Registry，即 `BodyParserRegistry`。
    原因：避免业务代码里到处 `if schemaVersion == ...`。

82. **`BodyParserRegistry` 放哪层？**
    结论：放 domain/application 可用的纯逻辑层；实现不依赖 infrastructure。
    原因：正文格式解析是业务内容规则，不是 Mongo 技术细节。

83. **正文校验错误如何返回？**
    结论：返回路径级 `details`，使用英文机器码和可选 `messageKey`，最多 20 个。
    原因：前端需要定位具体 block；协议不能写死中文文案；上限防止错误响应过大。

84. **这些结论是否写回文档、图和 ADR？**
    结论：需要写回服务专题文档、Content 图、错误码、CONTEXT 和 ADR。
    原因：否则后续实现会回到旧 Saga 误解，且无法复盘每个取舍。

## 后续修正记录

- 本轮一开始使用了 `grillme`，后来确认应使用 `grill-with-docs`；已补 Content `CONTEXT.md` 和模块内 ADR。
- ADR 最初曾落到全局 `docs/adr/`，后来修正到 `docs/architecture/services/content/adr/`，因为这些是 Content 上下文内决策。
- `draft_body_id` 最初曾考虑在一个 draft 生命周期内复用；后来因 Mongo 成功、PG 失败的不一致风险，修正为草稿 copy-on-write。
