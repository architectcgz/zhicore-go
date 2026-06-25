# Context Lifecycle and Timeout Budget

Use `context.Context` for lifecycle control and infrastructure metadata. Do not use it as a hidden business parameter bag.

## Context Values

Bad: business code depends on raw context keys and hidden values.

```go
func (r *ArticleRepo) Find(ctx context.Context, pageSize, pageNum int) ([]Article, error) {
	userID := ctx.Value("userID").(int64)
	tenantID := ctx.Value("tenantID").(string)
	skipAudit := ctx.Value("skip_audit").(bool)

	return r.findForUser(ctx, userID, tenantID, pageSize, pageNum, skipAudit)
}
```

Problems:

- callers cannot see the real inputs from the function signature
- key spelling and type changes are not checked by the compiler
- repositories now know about request/user/audit concerns they do not own

Better: keep context keys centralized and limited to infrastructure metadata.

```go
package infractx

import "context"

type requestIDKey struct{}
type principalKey struct{}

type Principal struct {
	ActorID  int64
	TenantID string
	Roles    []string
}

func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, id)
}

func RequestID(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(requestIDKey{}).(string)
	return id, ok
}

func WithPrincipal(ctx context.Context, p Principal) context.Context {
	return context.WithValue(ctx, principalKey{}, p)
}

func PrincipalFrom(ctx context.Context) (Principal, bool) {
	p, ok := ctx.Value(principalKey{}).(Principal)
	return p, ok
}
```

At the transport/application boundary, convert infrastructure metadata into explicit business input. Keep the handler as an HTTP adapter: decode, call the use case, write the response.

```go
type Actor struct {
	ID       int64
	TenantID string
	Roles    []string
}

type ListArticlesQuery struct {
	Actor    Actor
	PageSize int
	PageNum  int
	Sort     string
}

func (h *ArticleHTTPHandler) ListArticles(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	principal, ok := infractx.PrincipalFrom(ctx)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	actor := Actor{
		ID:       principal.ActorID,
		TenantID: principal.TenantID,
		Roles:    principal.Roles,
	}
	query := ListArticlesQuery{
		Actor:    actor,
		PageSize: parsePageSize(r),
		PageNum:  parsePageNum(r),
		Sort:     r.URL.Query().Get("sort"),
	}

	articles, err := h.listArticles.Execute(ctx, query)
	if err != nil {
		h.writeError(w, err)
		return
	}

	h.writeJSON(w, articles)
}
```

The use case and repository keep business values explicit.

```go
func (uc *ListArticlesUseCase) Execute(ctx context.Context, query ListArticlesQuery) ([]Article, error) {
	if query.PageSize <= 0 {
		query.PageSize = 20
	}

	return uc.repo.List(ctx, ArticleListFilter{
		TenantID: query.Actor.TenantID,
		PageSize: query.PageSize,
		PageNum:  query.PageNum,
		Sort:     query.Sort,
	})
}
```

## Timeout Budget Ownership

Whoever owns the business rhythm owns the timeout budget. Use-case/application orchestration sets the total budget for the business operation. The default is to pass that same context through the required steps, not to pre-allocate a timeout for every DB, cache, RPC, or queue call. A handler may enforce an HTTP transport cap, but it should not own business dependency budgets.

Timeout values must come from named configuration or injected policy. For the `Config` / `Default` / `Load` split, read `references/configuration-defaults-and-loading.md`.

Bad: a repository invents an unrelated timeout.

```go
func (r *ArticleRepo) List(ctx context.Context, filter ArticleListFilter) ([]Article, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	rows, err := r.db.QueryContext(ctx, listArticlesSQL, filter.TenantID, filter.PageSize, filter.PageNum)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanArticles(rows)
}
```

The repository does not know whether the caller has 80ms, 800ms, or 8s left.

Better: load and validate named timeout configuration at the application boundary, inject narrow timeout settings into the use case, and keep the handler as a transport adapter. It may pass `r.Context()` directly or apply a transport-level cap. The use case owns the business operation budget.

```go
func (h *ArticleHTTPHandler) CreateArticle(w http.ResponseWriter, r *http.Request) {
	cmd, err := h.decodeCreateArticleCommand(r)
	if err != nil {
		h.writeError(w, err)
		return
	}

	article, err := h.createArticle.Execute(r.Context(), cmd)
	if err != nil {
		h.writeError(w, err)
		return
	}

	h.writeJSON(w, article)
}
```

The use case applies the total business budget. Repository and client calls consume the deadline they receive unless a dependency has an explicit protective cap.

```go
func (uc *CreateArticleUseCase) Execute(parent context.Context, cmd CreateArticleCommand) (*Article, error) {
	ctx, cancel := context.WithTimeout(parent, uc.timeouts.Create)
	defer cancel()

	article, err := uc.repo.Create(ctx, ArticleCreate{
		TenantID: cmd.Actor.TenantID,
		AuthorID: cmd.Actor.ID,
		Title:    cmd.Input.Title,
		Body:     cmd.Input.Body,
	})
	if err != nil {
		return nil, err
	}

	if err := uc.search.IndexArticle(ctx, article.ID); err != nil {
		return nil, err
	}

	return article, nil
}
```

Repository code consumes the budget it receives instead of defining its own business timeout.

```go
func (r *ArticleRepo) Create(ctx context.Context, input ArticleCreate) (*Article, error) {
	row := r.db.QueryRowContext(ctx, createArticleSQL, input.TenantID, input.AuthorID, input.Title, input.Body)

	var article Article
	if err := row.Scan(&article.ID, &article.TenantID, &article.AuthorID, &article.Title, &article.Body); err != nil {
		return nil, err
	}
	return &article, nil
}
```

Local tightening is acceptable for a dependency with a known SLA or risk profile, but it is an explicit protective cap, not the default shape for every step. The timeout still comes from client configuration.

```go
func (c *BillingClient) Charge(ctx context.Context, req ChargeRequest) (*ChargeResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.chargeTimeout)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint+"/charges", encode(req))
	if err != nil {
		return nil, err
	}

	return c.do(httpReq)
}
```

## Goroutine Contexts

Bad: request context is passed into work that must continue after the HTTP request returns.

```go
func (h *ArticleHTTPHandler) CreateArticle(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	article, err := h.createArticle.Execute(ctx, commandFrom(r))
	if err != nil {
		h.writeError(w, err)
		return
	}

	go h.cache.InvalidateArticleList(ctx, article.ID)
	go h.notifier.SendNewArticleNotification(ctx, article.ID)

	w.WriteHeader(http.StatusCreated)
}
```

After the handler returns, `r.Context()` is canceled. The goroutines may fail immediately or halfway through an I/O call.

For best-effort side effects, create an independent bounded context and copy only infrastructure metadata that is still useful.

```go
func (uc *CreateArticleUseCase) scheduleCacheInvalidation(reqCtx context.Context, articleID int64) {
	requestID, _ := infractx.RequestID(reqCtx)

	go func(articleID int64, requestID string) {
		ctx, cancel := context.WithTimeout(context.Background(), uc.timeouts.Cache)
		defer cancel()

		if requestID != "" {
			ctx = infractx.WithRequestID(ctx, requestID)
		}

		if err := uc.cache.InvalidateArticleList(ctx, articleID); err != nil {
			uc.logger.WarnContext(ctx, "invalidate article cache", "article_id", articleID, "error", err)
		}
	}(articleID, requestID)
}
```

For critical side effects, enqueue durable work from the use case instead of relying on a bare goroutine in the handler.

```go
type OutboxJob struct {
	Type      string
	ArticleID int64
	RequestID string
}

func (uc *CreateArticleUseCase) enqueueArticleNotification(ctx context.Context, articleID int64) error {
	requestID, _ := infractx.RequestID(ctx)
	return uc.outbox.Enqueue(ctx, OutboxJob{
		Type:      "article.notification",
		ArticleID: articleID,
		RequestID: requestID,
	})
}
```

Long-lived workers should use an application lifecycle context. Each task gets its own timeout.

```go
func (w *Worker) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case task, ok := <-w.tasks:
			if !ok {
				return nil
			}
			w.processOne(ctx, task)
		}
	}
}

func (w *Worker) processOne(ctx context.Context, task Task) {
	taskCtx, cancel := context.WithTimeout(ctx, task.Timeout)
	defer cancel()

	if err := w.processor.Process(taskCtx, task); err != nil {
		w.logger.ErrorContext(taskCtx, "process task", "task_id", task.ID, "error", err)
	}
}
```

## Review Checklist

- Does every hidden `ctx.Value` dependency deserve to be explicit input instead?
- Are context keys centralized with typed accessors instead of raw strings?
- Is auth/principal data converted to an explicit actor before business logic spreads out?
- Does the code owner that controls the business flow also own the timeout budget?
- Are timeout budgets named, validated, and injected instead of hard-coded as magic duration literals?
- Does the code avoid mandatory per-step timeout allocation when one total business budget is enough?
- When a child deadline is used, is it a deliberate dependency cap that only tightens the parent deadline?
- Can post-request goroutines outlive request cancellation safely?
- Do critical side effects survive process crashes through durable queueing, MQ, or outbox persistence?
