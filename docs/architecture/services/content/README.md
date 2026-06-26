# Content 服务设计

## 事实来源

- Java `zhicore-content` controller：Post command/query、like/favorite、tag、admin、outbox、reader presence。
- Java `content-service-design.md`、`content-visibility-and-projection-evolution.md`、`post-reading-presence.md`。
- Java `zhicore-content/src/main/resources/db/schema.sql`。
- `zhicore-client` 和 `zhicore-integration` 中 post 事件与 DTO。

## 职责边界

`zhicore-content` 拥有文章主数据、文章发布生命周期、标签、分类、话题引用、文章互动写模型、文章统计、作者快照和内容服务内部投影。

Content 不拥有用户资料事实、评论树、搜索索引、热榜分数或通知收件箱。

## DDD 目标设计

Content 是独立限界上下文。统一语言以”文章、草稿、正文、发布、定时发布、删除、恢复、标签、分类、话题引用、点赞、收藏、统计、作者快照、内部投影”为核心，不把 User、Upload、Comment、Search、Ranking 或 Notification 的模型引入 Content 领域层。

DDD 设计用于指导 Go 目标实现，不表示当前 Go 代码已经完成。Java 侧已有领域模型和命令服务是事实来源之一，但 Go 实现按本仓库 `api/http -> application -> domain/ports -> infrastructure` 的依赖方向重新落点。

### DDD 战术模式应用

本服务应用以下 DDD 战术模式：

- **聚合（Aggregate）**：`Post`、`Tag`、`Category`、`PostStats`、`PostEngagement`
- **值对象（Value Object）**：封装业务概念的不可变对象，避免原始类型偏执
- **领域服务（Domain Service）**：不自然归属于单个实体的业务规则，不依赖基础设施
- **领域事件（Domain Event）**：聚合内发生的业务事实，用于触发后续流程
- **工厂（Factory）**：封装复杂聚合的创建逻辑，确保创建时的不变量
- **仓储（Repository）**：聚合的持久化抽象，隔离基础设施
- **应用服务（Application Service）**：编排用例流程，拥有事务边界
- **端口（Port）**：领域层和应用层对基础设施的抽象依赖

### 限界上下文与子域

Content 上下文内按职责拆成以下子域：

| 子域 | 职责 | 主要存储 |
| --- | --- | --- |
| Post Lifecycle | 文章创建、元数据修改、发布、定时发布、撤回、删除、恢复、归档和作者快照更新 | `posts`、`scheduled_publish_event` |
| Post Body / Draft | 正文、草稿、富文档块、媒体引用和 PostgreSQL / MongoDB 写入状态 | MongoDB 文档、`posts.write_state` |
| Tag / Category | 标签、分类、slug、标签关系和标签统计投影 | `tags`、`categories`、`post_tags`、`tag_stats` |
| Engagement | 点赞、取消点赞、收藏、取消收藏和文章本地统计 | `post_likes`、`post_favorites`、`post_stats` |
| Projection / Integration | 服务内投影任务、跨服务 outbox、消费幂等和管理端重试 | `domain_event_task`、`outbox_event`、`consumed_events`、`outbox_retry_audit` |
| Reader Presence | 读者在线 presence session、离开和在线状态查询 | Redis |

### 聚合

#### `Post` 聚合

`Post` 是文章主数据聚合根，负责维护文章生命周期和强一致状态：

- **标识**：内部 `PostID`（数据库生成）；如对外需要短公开标识，使用独立 `PublicPostID` 字段，不依赖 `zhicore-id-generator`。
- **归属**：`OwnerID` 引用 User；`OwnerSnapshot` 是 Content 本地快照（见下方一致性说明），不是用户资料事实源。
- **元数据**：标题、摘要、封面文件引用、话题引用、标签 ID 集合、发布时间、定时发布时间、归档标记、乐观锁版本。
- **状态**：`Draft`、`Published`、`Scheduled`、`Deleted`。
- **行为**：`UpdateMeta`、`UpdateTags`、`SetTopic`、`SetCoverImage`、`Publish`、`SchedulePublish`、`ExecuteScheduledPublish`、`Unpublish`、`CancelSchedule`、`Delete`、`Restore`、`MarkArchived`、`UpdateOwnerSnapshot`。
- **领域事件**：聚合行为会产生领域事件（如 `PostPublished`、`PostDeleted`），由 application 层转换为集成事件。

`Post` 聚合内只放必须强一致的文章状态规则。**浏览量、点赞数、收藏数、评论数等高频计数不作为 `Post` 聚合的一部分**，它们属于独立的 `PostStats` 聚合，避免把文章聚合变成热点聚合。

**`OwnerSnapshot` 的一致性语义**：

- `OwnerSnapshot`（作者昵称、头像、资料版本）是**非强一致的便利投影**，不参与 `Post` 的核心业务规则校验。
- 它存储在 `Post` 聚合内是为了避免查询文章列表时大量跨服务调用 User 服务。
- 由异步事件 `user.profile.updated` 驱动更新，允许短暂的最终一致性。
- 作者快照只能用更新版本覆盖旧版本，乱序事件不能回滚快照（版本号单调递增）。

`Post` 的核心不变量：

- 已删除文章不能编辑、发布、定时发布或更新标签。
- 只有草稿文章可以设置定时发布。
- 只有定时发布文章可以取消定时或执行定时发布。
- 已发布文章不能重复发布。
- 发布前必须满足标题等元数据规则；**正文是否存在由 application 在编排 Saga 时校验**，不属于 `Post` 聚合的职责（见下方 Saga 说明）。

#### `PostBody` / `DraftSnapshot`

`PostBody` 和 `DraftSnapshot` 是围绕 `PostID` 的内容值对象或文档对象，主要表达正文、草稿、内容类型、富文档块和媒体引用。

它们不单独成为跨事务聚合根。创建、保存草稿、更新正文和发布由 application Saga 编排（见下方 Saga 说明）。

MongoDB 写入失败时，application 必须通过补偿任务让文章进入可恢复状态，**但补偿状态不作为 `Post` 聚合的领域字段**，而是由 infrastructure 层的 Saga 协调器管理。

#### `Tag` 聚合

`Tag` 是标签聚合根：

- `TagID` 是内部标识。
- `TagName` 是展示名称。
- `TagSlug` 是全局唯一自然键，创建后不可变。
- 描述可以修改。

`Tag` 只负责标签自身规则，不拥有文章列表。标签下文章由 `post_tags` 关系和查询模型提供。

#### `Category`

`Category` 是分类聚合根或受控参考数据：

- 维护名称、slug、描述、父分类和排序。
- Go 第一阶段如果没有分类管理 API，可以先作为只读参考数据实现。
- `Post` 只保存分类或话题引用，不把分类树嵌入文章聚合。

#### `PostStats` 聚合

`PostStats` 是**独立的统计聚合根**，不属于 `Post` 聚合，负责维护文章的高频计数：

- **标识**：`PostID`（与文章一对一关系）
- **统计字段**：`ViewCount`、`LikeCount`、`FavoriteCount`、`CommentCount`
- **行为**：`IncrementViews`、`IncrementLikes`、`DecrementLikes`、`IncrementFavorites`、`DecrementFavorites`、`UpdateCommentCount`
- **不变量**：所有计数不能为负数

**为什么 `PostStats` 是独立聚合**：

- 点赞、收藏、评论等高频操作如果修改 `Post` 聚合，会导致严重的并发冲突和乐观锁重试。
- 统计计数的一致性要求低于文章状态（允许短暂不准确），不需要和 `Post` 强耦合。
- 独立聚合后，点赞/收藏事务只修改 `PostStats` 和关系表，不锁 `Post` 聚合。

**事务边界示例**（点赞）：

```text
单个事务：
  post_likes 表（插入）
  + post_stats.like_count（原子 +1）
  + outbox_event（集成事件）
```

注意：这个事务**不涉及 `Post` 聚合**，`Post` 聚合的版本号不会因点赞而递增。

#### `PostEngagement`

点赞和收藏不是聚合根，它们是以 `(PostID, UserID)` 为自然唯一键的互动关系实体：

- `PostLike`：用户对文章的点赞关系
- `PostFavorite`：用户对文章的收藏关系

这些关系不独立存在，总是和 `PostStats` 聚合一起修改。写入时由 application 在同一 PostgreSQL 事务内完成：

```text
post_likes / post_favorites 表（插入/删除关系）
+ PostStats 聚合（修改计数）
+ outbox_event（集成事件）
```

Redis 点赞/收藏状态和计数缓存只在事务提交后 best-effort 更新，失败不回滚业务事务。

#### 非领域聚合

以下对象不建成领域聚合：

- `outbox_event`：跨服务集成事件投递台账，属于 infrastructure 可靠消息机制。
- `domain_event_task`：Content 服务内投影任务，属于 infrastructure 异步任务调度。
- `consumed_events`：消费幂等记录，属于 integration 运行机制。
- Reader presence：短生命周期 Redis 状态，不作为持久化领域聚合。
- `scheduled_publish_event`：定时发布任务记录，属于 infrastructure 调度机制。
- `outbox_retry_audit`：管理端重试审计，属于运维数据，不是领域模型。

### 领域事件

领域事件是聚合内发生的业务事实，由聚合根产生，application 层消费并转换为集成事件。

**核心领域事件**：

| 领域事件 | 触发聚合行为 | 业务含义 |
| --- | --- | --- |
| `PostCreated` | `Post.Create()` | 文章草稿创建 |
| `PostPublished` | `Post.Publish()` | 文章发布，内容对外可见 |
| `PostUnpublished` | `Post.Unpublish()` | 文章撤回，回到草稿状态 |
| `PostDeleted` | `Post.Delete()` | 文章软删除，不再可见 |
| `PostRestored` | `Post.Restore()` | 文章从删除状态恢复 |
| `PostMetaUpdated` | `Post.UpdateMeta()` | 标题、摘要、封面等元数据更新 |
| `PostTagsUpdated` | `Post.UpdateTags()` | 文章标签关系变更 |
| `OwnerSnapshotRefreshed` | `Post.UpdateOwnerSnapshot()` | 作者快照版本更新 |
| `PostLiked` | `PostStats.IncrementLikes()` | 用户点赞 |
| `PostUnliked` | `PostStats.DecrementLikes()` | 用户取消点赞 |
| `PostFavorited` | `PostStats.IncrementFavorites()` | 用户收藏 |
| `PostUnfavorited` | `PostStats.DecrementFavorites()` | 用户取消收藏 |
| `TagCreated` | `Tag.Create()` | 标签创建 |
| `PostTagAssociated` | `Post.UpdateTags()` | 文章关联标签 |

**领域事件的生命周期**：

```text
1. 聚合根行为产生领域事件
   post.Publish() -> post.events.append(PostPublished{...})

2. Application 保存聚合后收集事件
   events := post.PopEvents()

3. Application 转换为集成事件并写入 outbox
   for event := range events {
       integrationEvent := mapper.ToIntegrationEvent(event)
       outbox.Append(integrationEvent)
   }

4. Infrastructure dispatcher 投递集成事件到 RabbitMQ
   DispatchOutboxEvents()

5. 其他服务消费集成事件
   Ranking、Notification、Search 等
```

**领域事件与集成事件的区别**：

- **领域事件**：领域层的纯业务概念，不依赖 JSON、Protobuf、RabbitMQ 等技术实现。
- **集成事件**：跨服务的契约消息，有明确的 schema、routing key、envelope 格式。

领域事件可以在 domain 层测试中直接断言，不需要启动 RabbitMQ。

### 值对象

Content 领域层优先用值对象表达有业务含义的基础值，避免在业务规则里到处传裸 `string` / `int64`。

**为什么使用封装的 ID 而不是裸类型**：

1. **类型安全**：`func GetPost(postID PostID, userID UserID)` 比 `func GetPost(a int64, b int64)` 更安全，编译器能防止参数传错。
2. **表达领域概念**：类型名称本身就是文档，`PostID` 一眼就能看出是文章标识。
3. **封装验证逻辑**：`NewPostID(id int64)` 可以校验 ID 合法性，避免到处重复校验代码。
4. **防止不同 ID 体系混淆**：`PostID`（数据库 ID）vs `PublicPostID`（对外短 ID）不会误用。
5. **避免原始类型偏执**：消除代码中大量 `// ownerID 是用户 ID，不是文章 ID` 这样的注释。

**核心值对象**：

| 值对象 | 含义 | 示例 |
| --- | --- | --- |
| `PostID`、`PublicPostID` | 文章内部标识和可选外部公开标识 | `PostID(12345)`、`PublicPostID("a3x9k")` |
| `UserID`、`OwnerID` | 作者或操作者引用 | `UserID(67890)` |
| `TagID`、`TagName`、`TagSlug` | 标签标识、名称和唯一 slug | `TagSlug("golang-concurrency")` |
| `CategoryID`、`TopicID` | 分类和话题引用 | `CategoryID(3)` |
| `FileID` | Upload 拥有的文件引用，例如封面或正文媒体 | `FileID("f_abc123")` |
| `PostTitle`、`PostExcerpt` | 标题和摘要，封装长度、空值和摘要生成规则 | `PostTitle.New("Go 并发模式")` |
| `PostStatus` | 文章生命周期状态 | `Draft`、`Published`、`Scheduled`、`Deleted` |
| `OwnerSnapshot` | 作者昵称、头像文件引用和资料版本快照 | `OwnerSnapshot{Name, AvatarID, Version}` |
| `PostContent`、`ContentBlock`、`MediaResource` | 正文、富文档块和媒体引用 | 富文档模型或纯文本 |
| `ScheduledAt`、`PublishedAt` | 发布相关时间点 | `time.Time` 封装 |
| `PostStats` | 文章统计计数值，计数不能为负 | `ViewCount(1234)` |
| `CursorToken` | 列表游标的编码值，不把内部排序字段直接暴露给调用方 | `CursorToken("eyJ...base64...")` |

内部主键默认由各服务 PostgreSQL identity / sequence 生成。Go 目标设计不为普通文章、标签、点赞或收藏关系引入中心发号服务。

### 领域服务

领域服务只承载不自然属于单个实体或值对象的业务规则，**不依赖基础设施**（不依赖 HTTP、数据库、Redis、MQ）。

**纯业务规则的领域服务**：

| 领域服务 | 职责 |
| --- | --- |
| `PostPublishPolicy` | 判断 `Post` 聚合状态是否允许发布、定时发布或撤回（只检查聚合内字段，不查询外部） |
| `ExcerptGenerator` | 从正文生成摘要，去除 HTML 和多余空白，控制最大长度 |
| `TagSlugPolicy` | 校验标签名称并生成规范化 slug；拼音转换通过注入的 `SlugTransliterator` 接口实现 |
| `OwnerSnapshotPolicy` | 比较作者资料版本，决定是否允许刷新作者快照（版本单调递增检查） |
| `EngagementPolicy` | 判断文章是否可点赞或收藏（文章存在且为已发布状态） |
| `InternalEventPriorityPolicy` | 为 Content 内部事件分级，例如删除、恢复、发布是可见性收敛高优先级事件 |

**依赖倒置示例**（拼音转换）：

```go
// domain 层定义接口
type SlugTransliterator interface {
    Transliterate(text string) string
}

// domain 服务依赖抽象
type TagSlugPolicy struct {
    transliterator SlugTransliterator
}

func (p *TagSlugPolicy) GenerateSlug(tagName string) (TagSlug, error) {
    normalized := p.transliterator.Transliterate(tagName)
    // 业务规则：小写、替换空格为连字符、限制长度
    slug := strings.ToLower(strings.ReplaceAll(normalized, " ", "-"))
    if len(slug) > 50 {
        return TagSlug{}, ErrSlugTooLong
    }
    return TagSlug{value: slug}, nil
}

// infrastructure 层实现
type PinyinTransliterator struct {
    // 使用具体的拼音库
}
```

**基础设施检查由 Application 编排**：

例如"正文是否存在"需要查询 MongoDB，这不属于领域服务职责，而是 application 用例在调用 `Post.Publish()` 前的前置检查：

```go
// Application 层
type PublishPostUseCase struct {
    postRepo  PostRepository
    bodyStore PostBodyStore
    policy    PostPublishPolicy  // 只检查聚合状态
    clock     Clock
}

func (uc *PublishPostUseCase) Execute(ctx context.Context, cmd PublishPostCommand) error {
    post := uc.postRepo.Load(cmd.PostID)

    // 1. 领域规则检查（纯内存）
    if err := uc.policy.CanPublish(post); err != nil {
        return err
    }

    // 2. 基础设施检查（查询 MongoDB）
    if !uc.bodyStore.Exists(ctx, post.ID()); err != nil {
        return ErrBodyNotFound
    }

    // 3. 执行发布
    post.Publish(uc.clock.Now())
    uc.postRepo.Save(ctx, post)

    return nil
}
```

事务编排、调用 User / Upload、写 outbox、写 Mongo、更新 Redis、记录告警和返回 DTO 都属于 application 或 infrastructure，不放入领域服务。

### 工厂

工厂负责封装复杂聚合的创建逻辑，确保创建时的不变量。

**`PostFactory`**：

```go
type PostFactory struct {
    clock Clock
}

func (f *PostFactory) CreateDraft(
    ownerID UserID,
    title PostTitle,
    ownerSnapshot OwnerSnapshot,
) (*Post, error) {
    // 校验前置条件
    if title.IsEmpty() {
        return nil, ErrTitleRequired
    }
    if ownerSnapshot.IsZero() {
        return nil, ErrOwnerSnapshotRequired
    }

    now := f.clock.Now()

    post := &Post{
        id:            PostID(0),  // 数据库生成
        ownerID:       ownerID,
        title:         title,
        status:        Draft,
        ownerSnapshot: ownerSnapshot,
        createdAt:     now,
        updatedAt:     now,
        version:       1,
        events:        []DomainEvent{
            PostCreated{
                PostID:    PostID(0),  // repository 保存后回填
                OwnerID:   ownerID,
                CreatedAt: now,
            },
        },
    }

    return post, nil
}
```

**`TagFactory`**：

```go
type TagFactory struct {
    slugPolicy TagSlugPolicy
}

func (f *TagFactory) CreateTag(name TagName) (*Tag, error) {
    if name.IsEmpty() {
        return nil, ErrTagNameRequired
    }

    slug, err := f.slugPolicy.GenerateSlug(name.String())
    if err != nil {
        return nil, err
    }

    tag := &Tag{
        id:     TagID(0),  // 数据库生成
        name:   name,
        slug:   slug,
        events: []DomainEvent{
            TagCreated{
                TagID: TagID(0),
                Name:  name,
                Slug:  slug,
            },
        },
    }

    return tag, nil
}
```

工厂确保聚合创建时就满足业务规则，application 层直接使用工厂而不是手动拼装聚合。

### Application 用例

Content application 层按命令、查询分层组织 use case。application 拥有事务边界、权限上下文、幂等、端口调用和错误映射。

**命令用例（Commands）**：

- `CreatePost`：创建文章草稿，保存作者快照，初始化统计，必要时写正文/草稿和内部投影任务。
- `UpdatePostMeta`：更新标题、摘要、封面、话题和分类引用。
- `UpdatePostContent`：更新正文，由 Saga 协调 PostgreSQL 和 MongoDB 写入。
- `SaveDraft` / `DeleteDraft`：保存或删除草稿快照。
- `PublishPost`：校验作者、状态和正文，发布文章，写跨服务 outbox 和内部投影任务。
- `UnpublishPost`：撤回已发布文章，回到草稿状态并写投影任务。
- `SchedulePost` / `CancelSchedule` / `ExecuteScheduledPublish`：维护定时发布记录和最终发布。
- `DeletePost` / `RestorePost` / `PurgePost`：软删除、恢复和清理文章，删除/恢复事件属于内部投影 P0。
- `UpdatePostTags` / `RemovePostTag`：维护文章标签关系，触发标签统计投影。
- `LikePost` / `UnlikePost`：维护点赞关系、`PostStats` 统计和互动事件。
- `FavoritePost` / `UnfavoritePost`：维护收藏关系、`PostStats` 统计和互动事件。
- `SyncAuthorSnapshot`：消费用户资料更新事件，按版本刷新作者快照。
- `UpdateCommentCount`：消费评论事件，更新 `PostStats` 评论计数。

**查询用例（Queries）**：

- `GetPostDetail`、`GetPostContent`、`GetDraft`。
- `ListPublishedPosts`、`ListAuthorPosts`、`ListMyPosts`、`CursorListPosts`、`BatchGetPosts`。
- `GetPostTags`、`GetTagDetail`、`SearchTags`、`ListHotTags`、`ListPostsByTag`。
- `GetLikeStatus`、`BatchGetLikeStatus`、`GetLikeCount`。
- `GetFavoriteStatus`、`BatchGetFavoriteStatus`、`GetFavoriteCount`。
- `GetReaderPresence`。
- `ListAdminPosts`、`ListOutboxDeadOrFailedEvents`。

**命令和查询分离**：

- 命令用例修改状态，返回简单成功/失败或聚合 ID。
- 查询用例只读取，返回 DTO 或视图模型。
- 不在命令用例里嵌套复杂查询逻辑。

### Saga 协调器

PostgreSQL 与 MongoDB 的跨存储写入**不伪装成强事务**，由 infrastructure 层的 Saga 协调器处理补偿逻辑。

**`PostPublishSaga` 示例**：

```go
// infrastructure/saga/post_publish_saga.go
type PostPublishSaga struct {
    postRepo      PostRepository
    bodyStore     PostBodyStore
    outbox        OutboxStore
    taskStore     CompensationTaskStore
    txRunner      TransactionRunner
}

func (s *PostPublishSaga) Execute(ctx context.Context, cmd PublishPostCommand) error {
    return s.txRunner.RunInTransaction(ctx, func(txCtx context.Context) error {
        // 1. 更新 PostgreSQL：Post 聚合发布
        post := s.postRepo.Load(txCtx, cmd.PostID)
        post.Publish(time.Now())
        s.postRepo.Save(txCtx, post)

        // 2. 收集领域事件并转换为集成事件
        events := post.PopEvents()
        for _, event := range events {
            integrationEvent := toIntegrationEvent(event)
            s.outbox.Append(txCtx, integrationEvent)
        }

        // 3. PostgreSQL 事务提交
        return nil
    })

    // 4. 事务提交后，尝试写 MongoDB（不在事务内）
    if err := s.bodyStore.PublishBody(ctx, cmd.PostID); err != nil {
        // MongoDB 写入失败，创建补偿任务（异步重试）
        s.taskStore.CreateCompensationTask(ctx, CompensatePostBody{
            PostID:    cmd.PostID,
            Operation: "publish",
            Reason:    err.Error(),
        })
        // 不回滚 PostgreSQL，文章已发布，MongoDB 投影稍后补偿
    }

    return nil
}
```

**补偿任务处理**：

- MongoDB 写入失败不回滚 PostgreSQL 事务。
- 创建补偿任务记录到 `compensation_tasks` 表（或复用 `domain_event_task`）。
- 后台 worker 定期重试补偿任务。
- 查询接口以 PostgreSQL 的文章状态为准，MongoDB 只承载正文内容。

**关键原则**：

- `Post` 聚合不包含 `WriteState` / `IncompleteReason` 等技术补偿字段。
- Saga 协调器属于 infrastructure 层，不污染 domain 层。
- Application 命令用例调用 Saga，不直接处理双写逻辑。

### Ports

Ports 放在 `services/zhicore-content/internal/content/ports`，按聚合或用例族定义接口，避免过度碎片化。

**核心端口（按聚合分组）**：

| Port | 职责 | 说明 |
| --- | --- | --- |
| `PostRepository` | Post 聚合持久化 | 加载、保存、按作者校验所有权、乐观锁更新 |
| `PostQueryRepository` | Post 查询 | 详情、列表、批量、作者文章、管理端查询 |
| `PostStatsRepository` | PostStats 聚合持久化 | 初始化、原子增减计数、读取统计 |
| `PostContentStore` | 正文和草稿存储 | 保存/读取/删除 MongoDB 正文和草稿 |
| `TagRepository` | Tag 聚合持久化和查询 | 按 slug 查找、创建、批量查询 |
| `PostTagRepository` | 文章标签关系 | 替换、删除、批量查询文章标签 |
| `CategoryRepository` | 分类查询 | 查询分类或话题引用合法性 |
| `PostEngagementRepository` | 点赞/收藏关系 | 插入/删除/查询 `(post_id, user_id)` 关系 |

**基础设施机制端口**：

| Port | 职责 | 说明 |
| --- | --- | --- |
| `TransactionRunner` | 显式事务边界 | 避免 handler 或 repository 偷偷拥有业务事务 |
| `OutboxPublisher` | 跨服务事件发布 | 业务事务内追加 outbox 记录（不是独立的 `OutboxStore`） |
| `InternalEventPublisher` | 内部投影任务发布 | 业务事务内追加内部事件任务 |
| `ConsumedEventStore` | 消费幂等 | 记录消费过的事件 ID |
| `CompensationTaskStore` | Saga 补偿任务 | 创建和查询补偿任务 |

**缓存和外部服务端口**：

| Port | 职责 | 说明 |
| --- | --- | --- |
| `PostCacheStore` | Post 缓存 | cache-aside、失效、三态缓存 |
| `TagCacheStore` | Tag 缓存 | 热门标签缓存 |
| `EngagementCacheStore` | 点赞/收藏缓存 | Redis 状态和计数缓存 |
| `ReaderPresenceStore` | Presence 状态 | session、leave、presence 查询 |
| `UserProfileClient` | User 服务调用 | 获取作者摘要（创建文章和作者快照刷新） |
| `FileResourceClient` | Upload 服务调用 | 解析或清理文件引用 |
| `Clock` | 时间源 | 可测试的时间抽象 |

**端口设计原则**：

- **按聚合分组**：`PostRepository` 包含 Post 聚合的所有持久化方法，不拆成 10 个小接口。
- **查询分离**：读多写少的场景，`PostQueryRepository` 独立于 `PostRepository`。
- **基础设施机制不作为 domain/ports**：Outbox、InternalEventTask 的调度器和 dispatcher 属于 infrastructure，但发布接口（`OutboxPublisher`）可以作为端口供 application 使用。
- **避免宽泛接口**：不定义 `Store` 大接口包含所有 CRUD。

**不定义的端口**：

- `IdGeneratorClient`：内部 ID 由数据库生成，不依赖中心发号服务。
- `ScheduledPublishStore`：定时发布调度属于 infrastructure，不需要端口抽象（或合并到 `PostRepository`）。

### 一致性与事务边界

**文章命令事务**：

```text
单个 PostgreSQL 事务：
  posts 表（Post 聚合）
  + post_tags（标签关系）
  + outbox_event（集成事件）
  + domain_event_task（内部投影任务）
```

**点赞/收藏事务**：

```text
单个 PostgreSQL 事务：
  post_likes / post_favorites（关系表）
  + post_stats（PostStats 聚合，原子增减计数）
  + outbox_event（集成事件）

事务提交后：
  best-effort 更新 Redis 缓存（失败不回滚）
```

注意：点赞/收藏事务**不修改 `Post` 聚合**，`Post` 的乐观锁版本号不递增。

**标签事务**：

```text
单个 PostgreSQL 事务：
  tags（Tag 聚合）
  + post_tags（关系表）
  + domain_event_task（PostTagsUpdated，触发标签统计投影）
```

标签统计和热门标签缓存由内部投影任务最终一致更新。

**跨 PostgreSQL 与 MongoDB 的正文写入**：

不伪装成分布式事务，由 Saga 协调器处理：

1. PostgreSQL 事务提交（Post 聚合状态已变更）
2. 尝试写 MongoDB 正文
3. MongoDB 失败时，创建补偿任务，不回滚 PostgreSQL
4. 后台 worker 重试补偿任务
5. 查询接口以 PostgreSQL 状态为准，MongoDB 只是内容投影

**内部事件优先级**：

- **P0（高优先级）**：删除、恢复、发布、撤回，影响内容可见性收敛。
- **P1（普通优先级）**：标签统计、缓存失效、MongoDB 内容同步。

Dispatcher 优先处理高优先级任务，确保可见性变更快速生效。

### Go 包落点

目标目录：

```text
services/zhicore-content/
  api/http/              # HTTP 入站适配器
  internal/content/
    application/
      commands/          # 命令用例
      queries/           # 查询用例
    domain/
      post/              # Post 聚合、值对象、领域事件
      poststats/         # PostStats 聚合
      tag/               # Tag 聚合
      engagement/        # PostLike、PostFavorite 实体
      shared/            # 跨聚合的值对象和领域服务
      events/            # 领域事件定义
    ports/               # 端口接口定义
    infrastructure/
      postgres/          # PostgreSQL 适配器（实现 Repository）
      mongo/             # MongoDB 适配器（实现 PostContentStore）
      redis/             # Redis 适配器（实现 CacheStore）
      rabbitmq/
        consumers/       # RabbitMQ 消费者（user.profile.updated 等）
        publishers/      # 实现 OutboxPublisher
      saga/              # Saga 协调器（PostPublishSaga 等）
      jobs/              # 后台任务（OutboxDispatcher、InternalEventWorker）
      clients/           # 外部服务 HTTP client（User、Upload）
    runtime/
      module.go          # 依赖注入和模块装配
```

**分层依赖方向**：

```text
api/http -> application -> domain <- ports <- infrastructure
```

- `domain` 不依赖任何其他层
- `application` 依赖 `domain` 和 `ports`
- `infrastructure` 实现 `ports`，依赖 `domain`（聚合、值对象）
- `api/http` 依赖 `application`

**消费者、任务调度、Saga 归属**：

- **消费者**（`UserProfileUpdatedConsumer`）：`infrastructure/rabbitmq/consumers/`，调用 application 命令用例。
- **后台任务**（`OutboxDispatcher`、`InternalEventWorker`）：`infrastructure/jobs/`，不属于 application 用例。
- **Saga 协调器**（`PostPublishSaga`）：`infrastructure/saga/`，不属于领域服务。

第一版可以不机械拆出所有子包；如果代码量较小，`domain` 下可先保留少量文件。拆包标准是职责和依赖边界，而不是为了看起来像 DDD。

### 推荐首个实现切片

Content 第一轮学习和实现建议选”创建草稿并发布文章”，覆盖核心 DDD 模式：

**实施步骤**：

1. **Domain 层**：
   - 建 `Post` 聚合和值对象：`PostID`、`OwnerID`、`PostTitle`、`PostStatus`、`OwnerSnapshot`
   - 建 `PostFactory` 工厂
   - 建 `PostPublishPolicy` 领域服务
   - 定义 `PostCreated`、`PostPublished` 领域事件

2. **Ports 层**：
   - 定义 `PostRepository`、`PostContentStore`、`OutboxPublisher`、`UserProfileClient`、`Clock`、`TransactionRunner`

3. **Domain 测试**（用内存 fake）：
   - 测试 `Post.Publish()` 的不变量（草稿可发布，已发布不能重复发布）
   - 测试 `PostFactory.CreateDraft()` 的前置条件
   - 测试 `PostPublishPolicy` 的业务规则

4. **Application 层**：
   - 建 `CreatePost` 和 `PublishPost` 命令用例
   - 用内存 fake 实现端口，测试用例编排逻辑

5. **Infrastructure 层**：
   - 实现 PostgreSQL `PostRepository`
   - 实现 MongoDB `PostContentStore`
   - 实现 `PostPublishSaga` 协调器

6. **HTTP 层**：
   - 实现 `POST /api/v1/posts`（创建草稿）
   - 实现 `POST /api/v1/posts/{postId}/publish`（发布）

**这个切片能覆盖**：

- ✅ 聚合根、值对象、领域事件
- ✅ 工厂模式
- ✅ 领域服务
- ✅ 端口和适配器
- ✅ Application 用例编排
- ✅ Saga 协调器
- ✅ 事务边界
- ✅ Onion 依赖方向

同时不会一开始就把点赞、标签、投影、管理端全部卷入，降低学习曲线。

## API 保留范围

必须保留以下 API 族：

- `/api/v1/posts`：创建、更新、发布、取消发布、定时发布、删除、恢复、草稿、列表、详情、作者文章、游标和批量查询。
- `/api/v1/posts/{postId}/like`、`favorite`：点赞、取消点赞、收藏、取消收藏、状态和计数查询。
- `/api/v1/posts/{postId}/content`、`draft`：正文和草稿读取。
- `/api/v1/posts/{postId}/tags`：文章标签读写。
- `/api/v1/posts/{postId}/readers`：阅读 presence session、leave、presence 查询。
- `/api/v1/tags`：标签详情、列表、搜索、热门和标签文章。
- `/api/v1/admin/posts`：管理端文章查询和删除。
- `/api/v1/admin/outbox`：outbox dead/failed 查询和 retry。

## 数据归属

PostgreSQL：

- `posts`
- `post_stats`
- `post_likes`
- `post_favorites`
- `categories`
- `tags`
- `post_tags`
- `tag_stats`
- `scheduled_publish_event`
- `outbox_event`
- `domain_event_task`
- `outbox_retry_audit`
- `consumed_events`

MongoDB：

- 文章正文或富文档投影。

Redis：

- 文章详情缓存。
- 点赞/收藏状态和计数缓存。
- 阅读 presence 短生命周期状态。

作者快照字段如 `owner_name`、`owner_avatar_id`、`owner_profile_version` 保留为列，不默认改成 JSON。原因是这些字段需要参与补偿、查询、索引和精确更新。

## 主写流程

文章创建、编辑、发布、删除、恢复由 application use case 拥有事务边界：

```text
api/http -> application command -> postgres repository
                         -> domain_event_task
                         -> outbox_event
```

点赞/收藏必须在同一事务内完成：

- 关系表写入或删除。
- `post_stats.like_count` / `favorite_count` 原子增减。
- 对应 integration event 写入 `outbox_event`。

Redis 更新在事务提交后执行，失败不回滚业务事务。

## 事件

Content 生产：

- `content.post.published`
- `content.post.updated`
- `content.post.deleted`
- `content.post.tags.updated`
- `content.post.liked`
- `content.post.unliked`
- `content.post.favorited`
- `content.post.unfavorited`
- `content.post.viewed`

Content 消费：

- `user.profile.updated`：刷新作者快照。
- Comment 事件可以更新评论计数，但评论事实仍归 Comment。

关键跨服务事件统一走 producer outbox + RabbitMQ topic exchange。

## 跨服务依赖

- User：创建文章和刷新作者快照时读取用户资料摘要。
- Upload：文章图片、封面和正文资源只保存文件引用。
- Comment：文章详情可聚合评论计数，但不直接读评论库。

## Go 目标落点

- HTTP：`services/zhicore-content/api/http`
- Application：`services/zhicore-content/internal/content/application`
- Domain：`services/zhicore-content/internal/content/domain`
- Ports：`services/zhicore-content/internal/content/ports`
- Infrastructure：`postgres`、`redis`、`rabbitmq`、`mongo`、`clients`
- Runtime：`services/zhicore-content/internal/content/runtime/module.go`

## 迁移风险

- Java schema 来源存在漂移，Go migration 必须以服务归属重新整理，不原样复制全量初始化 SQL。
- 点赞/收藏链路必须保留 outbox 同事务语义，否则 Ranking/Notification 会丢事件。
- `domain_event_task` 是服务内投影，不应和跨服务 outbox 混用。
- Presence 只能作为附加能力，不能影响文章正文主链路。
- 避免把技术补偿状态（如 MongoDB 双写失败）泄漏到领域模型，使用 Saga 协调器隔离。
- `PostStats` 是独立聚合根，不属于 `Post` 聚合，确保点赞/收藏不锁文章聚合。

## 下一步

- 提取 Content 字段级 HTTP contract。
- 生成 Content migration 草案。
- 实现"创建草稿并发布文章"核心切片（见上方推荐首个实现切片）。
- 先写 domain 层测试（用内存 fake），验证聚合不变量和领域事件。
- 再写 application 层测试，验证用例编排逻辑。
- 最后接入 PostgreSQL / MongoDB / HTTP，完成端到端验证。

## DDD 设计总结

本文档按 DDD 战术模式重新设计 Content 服务，关键改进点：

1. **聚合边界清晰**：`Post`、`PostStats`、`Tag` 是独立聚合根，避免热点聚合和事务冲突。
2. **领域事件显式建模**：聚合行为产生领域事件，application 层转换为集成事件。
3. **值对象封装业务概念**：避免原始类型偏执，增强类型安全。
4. **领域服务不依赖基础设施**：纯业务规则，基础设施检查由 application 编排。
5. **工厂确保创建时不变量**：聚合创建逻辑封装在工厂，不散落在 application 层。
6. **Saga 协调器隔离技术复杂度**：PostgreSQL + MongoDB 双写补偿不污染领域模型。
7. **端口按聚合分组**：避免过度碎片化，降低依赖管理复杂度。
8. **命令查询分离**：写用例和读用例职责清晰。
9. **消费者和任务调度归属正确**：属于 infrastructure 层，不混入 application 用例。
10. **分层依赖方向符合 Onion Architecture**：domain 不依赖任何外层。
