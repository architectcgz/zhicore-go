# HTTP Handler 入出参参考形态

本文是 `api/http` handler 的代码参考形态。长期规则见 `docs/architecture/go-service-design.md`；本文只展示推荐写法，避免每个服务重复讨论 handler 如何 parse、validate、调用 use case 和写 response。

## 核心流水线

Handler 应该像一条清晰流水线：

```text
parse -> validate/map command -> call use case -> map response -> write
```

业务输入不要塞进 `context.Context`，也不要把 `*gin.Context`、`http.ResponseWriter` 传入 application、domain、ports 或 infrastructure。

## 推荐 endpoint 形态

```go
func (h *Handler) createPost(c *gin.Context) {
	ctx := c.Request.Context()
	w := c.Writer

	actor, err := actorFromRequest(c.Request)
	if err != nil {
		writeMappedError(w, err)
		return
	}

	var req createPostReq
	if err := decodeJSONBody(w, c.Request, &req); err != nil {
		writeDecodeError(w, err)
		return
	}

	cmd, err := createPostCommand(actor, req)
	if err != nil {
		writeValidationError(w)
		return
	}

	result, err := h.service.CreatePost(ctx, cmd)
	if err != nil {
		writeMappedError(w, err)
		return
	}

	sharedhttp.WriteSuccess(w, createPostResponse(result))
}
```

## Request helper 形态

Helper 默认只解析和返回错误，不直接写 response。这样 endpoint handler 能显式展示错误映射。

```go
func actorFromRequest(r *http.Request) (*application.Actor, error) {
	raw := strings.TrimSpace(r.Header.Get(userIDHeaderName))
	if raw == "" {
		return nil, errLoginRequired
	}

	userID, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || userID <= 0 {
		return nil, errLoginRequired
	}

	return &application.Actor{
		UserID: userID,
		Roles:  rolesFromRequest(r),
	}, nil
}

func postIDFromPath(c *gin.Context) (string, error) {
	postID := strings.TrimSpace(c.Param("postId"))
	if postID == "" {
		return "", application.ErrInvalidArgument
	}
	return postID, nil
}

func optionalPositiveIntQuery(c *gin.Context, key string) (int, error) {
	return sharedhttp.ParsePositiveInt(c.Query(key), 0, 0)
}

func decodeJSONBody(w http.ResponseWriter, r *http.Request, target any) error {
	return sharedhttp.DecodeJSONBodyLimited(w, r, maxJSONRequestBodyBytes, target)
}
```

`decodeJSONBody` 允许接收 `http.ResponseWriter`，因为 Go 标准库 `http.MaxBytesReader` 要求该参数。这个 `w` 仍然只属于 HTTP parsing 边界，不得继续传入业务层。

## Decode 错误映射

JSON decode 和 body limit 属于协议解析层错误。Handler 负责把 error 显式映射成服务自己的响应 envelope。

```go
func writeDecodeError(w http.ResponseWriter, err error) {
	if isRequestBodyTooLarge(err) {
		sharedhttp.WriteErrorCode(w, http.StatusRequestEntityTooLarge, 4015, "Body too large")
		return
	}
	writeValidationError(w)
}

func isRequestBodyTooLarge(err error) bool {
	var maxBytesErr *http.MaxBytesError
	return errors.As(err, &maxBytesErr)
}
```

推荐分类：

```text
empty body / malformed JSON / trailing JSON -> 400
body too large -> 413
valid JSON but validation failed -> service validation error
application/domain/ports semantic error -> writeMappedError
```

## Command / query 映射

复杂 endpoint 可以把 request DTO 到 application command/query 的映射抽成小函数。这个函数可以做字段级业务输入校验，但不要写 HTTP 响应。

```go
func createPostCommand(actor *application.Actor, req createPostReq) (application.CreatePostCommand, error) {
	if strings.TrimSpace(req.Title) == "" {
		return application.CreatePostCommand{}, application.ErrInvalidArgument
	}

	var body *application.PostBodyInput
	if req.Body != nil {
		body = &application.PostBodyInput{
			SchemaVersion: req.Body.SchemaVersion,
			Blocks:        req.Body.Blocks,
		}
	}

	return application.CreatePostCommand{
		Actor:       actor,
		Title:       req.Title,
		Summary:     req.Summary,
		CoverFileID: req.CoverFileID,
		TopicID:     req.TopicID,
		CategoryID:  req.CategoryID,
		Tags:        append([]string(nil), req.Tags...),
		Body:        body,
	}, nil
}
```

## Response mapper 形态

Application result 不直接等同于 HTTP response。Handler 层用 mapper 固定公开 contract 的时间、ID、枚举、nullable/optional 字段。

```go
func createPostResponse(result application.CreatePostResult) createPostResp {
	return createPostResp{
		PostID:      result.PostID,
		PostVersion: result.PostVersion,
	}
}

func formatTime(value time.Time) string {
	return sharedhttp.FormatRFC3339UTC(value)
}
```

## Context 边界

允许：

- cancellation
- deadline / timeout
- requestId / traceId
- narrow infrastructure metadata
- middleware 注入的认证 claims / principal metadata

不允许：

- `postId`
- `limit`
- `basePostVersion`
- request body 字段
- 业务分支所需的 actor/userID
- feature flag / dry-run 等业务参数

如果 middleware 把认证 claims 放进 `ctx`，handler 进入 application 前仍要转换成显式 `Actor` / `Identity`。
