# Runtime Scheduler and Bounded Concurrency

GMP is runtime knowledge, not an application architecture. Use it to reason about cost and failure modes, then write ordinary Go code with explicit ownership, cancellation, concurrency limits, and observability.

## Scope

Use this reference when reviewing or writing Go backend code that creates many goroutines, does fan-out work, runs CPU-heavy jobs, uses worker pools, shows goroutine leaks, or has scheduler-related symptoms in pprof.

Do not use GMP as a reason to hand-tune the scheduler in normal business code. Most backend code should express intent through `context`, `errgroup`, channels, semaphores, queues, worker pools, and validated config.

## Mental Model

- `G`: goroutine, the unit of Go work scheduled by the runtime.
- `M`: machine, an OS thread that executes Go or blocks in syscall/cgo/runtime work.
- `P`: processor, the runtime resource required to execute Go code. `GOMAXPROCS` controls how many `P`s can run Go code at the same time.

Practical consequences:

- Goroutines are cheap, not free. Unbounded goroutines still consume memory, scheduling work, file descriptors, DB connections, HTTP sockets, and downstream quota.
- More goroutines do not make CPU-bound work faster after available `P`s are saturated.
- Runtime scheduling order is not a correctness contract.
- Blocking network I/O is usually handled well by the runtime, but blocked dependencies still need timeout, cancellation, backpressure, and concurrency limits.
- cgo, long syscalls, `runtime.LockOSThread`, and CPU-heavy loops can change scheduler behavior enough that code-level limits and profiling matter.

## Rule

- Every goroutine must have an owner that can wait for it, cancel it, or deliberately detach it.
- Fan-out must have a named concurrency limit. The limit should come from validated config or injected policy.
- Return or record errors from concurrent work. Do not drop errors in fire-and-forget goroutines.
- CPU-bound concurrency should default near `runtime.GOMAXPROCS(0)` unless profiling proves another value.
- Do not rely on goroutine execution order, fairness, or `runtime.Gosched()` for correctness.
- Use `go test -race` for code with goroutines, shared state, maps, caches, worker pools, or background loops when the package can run under the race detector.

## Bad: Unbounded Fire-And-Forget Fan-Out

```go
func (s *IndexerService) ReindexAll(ctx context.Context) error {
	ids, err := s.repo.ListArticleIDs(ctx)
	if err != nil {
		return err
	}

	for _, id := range ids {
		go s.indexer.IndexArticle(ctx, id)
	}

	return nil
}
```

Problems:

- one goroutine per row can exhaust memory or downstream capacity
- the method returns before work finishes
- index errors are lost
- cancellation and lifecycle ownership are unclear
- request-scoped `ctx` may be reused for work that outlives the request

## Better: `errgroup` With A Configured Limit

Keep the concurrency budget explicit and validate it at wiring time.

```go
type ReindexConfig struct {
	Concurrency int
}

func (c ReindexConfig) Validate() error {
	if c.Concurrency <= 0 {
		return fmt.Errorf("reindex concurrency must be positive")
	}
	return nil
}

type IndexerService struct {
	repo        ArticleRepository
	indexer     SearchIndexer
	concurrency int
}

func NewIndexerService(repo ArticleRepository, indexer SearchIndexer, cfg ReindexConfig) (*IndexerService, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &IndexerService{
		repo:        repo,
		indexer:     indexer,
		concurrency: cfg.Concurrency,
	}, nil
}

func (s *IndexerService) ReindexAll(ctx context.Context) error {
	ids, err := s.repo.ListArticleIDs(ctx)
	if err != nil {
		return err
	}

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(s.concurrency)

	for _, id := range ids {
		id := id
		g.Go(func() error {
			return s.indexer.IndexArticle(ctx, id)
		})
	}

	return g.Wait()
}
```

This code makes the runtime contract visible:

- `SetLimit` caps active goroutines in this fan-out
- `WithContext` cancels siblings after the first error
- `Wait` gives ownership back to the caller
- the limit is named config, not a magic literal

## Ordered Results Without Shared Append

Do not append to a shared slice from goroutines. Preallocate result slots when order matters.

```go
func (s *ProfileService) LoadProfiles(ctx context.Context, userIDs []int64) ([]Profile, error) {
	profiles := make([]Profile, len(userIDs))

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(s.lookupConcurrency)

	for i, userID := range userIDs {
		i, userID := i, userID
		g.Go(func() error {
			profile, err := s.client.GetProfile(ctx, userID)
			if err != nil {
				return err
			}
			profiles[i] = profile
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}
	return profiles, nil
}
```

This keeps result order deterministic and avoids mutex-protected append logic. Each goroutine owns one result slot.

## CPU-Bound Work

For CPU-heavy work, a large goroutine count usually creates scheduling overhead instead of throughput. Resolve the worker count at the process boundary and inject it.

```go
type ImageConfig struct {
	Workers int
}

func ResolveImageWorkers(configured int) int {
	if configured > 0 {
		return configured
	}
	return runtime.GOMAXPROCS(0)
}

type ImageProcessor struct {
	workers int
}

func NewImageProcessor(cfg ImageConfig) (*ImageProcessor, error) {
	workers := ResolveImageWorkers(cfg.Workers)
	if workers <= 0 {
		return nil, fmt.Errorf("image workers must be positive")
	}
	return &ImageProcessor{workers: workers}, nil
}

func (p *ImageProcessor) ProcessBatch(ctx context.Context, jobs []ImageJob) error {
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(p.workers)

	for _, job := range jobs {
		job := job
		g.Go(func() error {
			if err := ctx.Err(); err != nil {
				return err
			}
			return p.processOne(ctx, job)
		})
	}

	return g.Wait()
}
```

Use a higher worker count only with measurement. CPU-bound defaults should not blindly match queue length or request size.

## Cancellation In Long CPU Loops

Modern Go has preemption, but cancellation is still an application contract. Long loops should observe `ctx` at predictable checkpoints.

```go
func CalculateScore(ctx context.Context, rows []ScoreRow) (int, error) {
	score := 0
	for i, row := range rows {
		if i%1024 == 0 {
			if err := ctx.Err(); err != nil {
				return 0, err
			}
		}

		score += expensiveScore(row)
	}
	return score, nil
}
```

The checkpoint interval should be chosen from work cost, not copied blindly.

## Do Not Use Scheduler Tricks As Synchronization

Bad: correctness depends on the scheduler eventually running another goroutine.

```go
func (c *Cache) WaitReady() {
	for !c.ready {
		runtime.Gosched()
	}
}
```

Problems:

- data race on `c.ready`
- no cancellation or timeout
- busy wait wastes CPU
- scheduler behavior becomes part of correctness

Better: synchronize with channels and context.

```go
type Cache struct {
	ready chan struct{}
}

func (c *Cache) WaitReady(ctx context.Context) error {
	select {
	case <-c.ready:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
```

Use `sync.Cond`, channels, mutexes, atomics, or context depending on the real contract. Do not use `runtime.Gosched()` to make races disappear.

## Thread-Affine Code

`runtime.LockOSThread` is rare in backend code. It belongs in narrow adapters for thread-affine integrations such as some cgo libraries, OS APIs, or driver runtimes. It should not appear in ordinary handlers, use cases, repositories, or workers.

If it is necessary:

- isolate it in one package
- document the external thread-affinity requirement
- keep the locked section small
- `defer runtime.UnlockOSThread()`
- expose a normal context-aware API to the rest of the application
- test shutdown behavior so a locked thread cannot strand work forever

## Observability Before Tuning

Do not tune `GOMAXPROCS`, worker counts, queue sizes, or scheduler assumptions by guess. Collect evidence first.

Useful checks:

```bash
go test -race ./path/...
go test -run TestName -count=100 ./path/...
go tool pprof http://localhost:6060/debug/pprof/profile
go tool pprof http://localhost:6060/debug/pprof/goroutine
go tool pprof http://localhost:6060/debug/pprof/block
go tool pprof http://localhost:6060/debug/pprof/mutex
GODEBUG=schedtrace=1000,scheddetail=1 ./your-server
```

Enable block and mutex profiles deliberately, usually behind diagnostic configuration:

```go
func EnableContentionProfiles() {
	runtime.SetBlockProfileRate(1)
	runtime.SetMutexProfileFraction(10)
}
```

Turn high-overhead profiling off when it is no longer needed.

## Review Checklist

- Is every goroutine owned by a caller, worker, queue, or lifecycle root?
- Is fan-out concurrency bounded by named config or injected policy?
- Does concurrent work return errors through `errgroup`, a result channel, or durable job state?
- Does CPU-bound work avoid using request size as goroutine count?
- Is `runtime.GOMAXPROCS(0)` used only to choose an execution budget, not as a magic fix?
- Does the code avoid relying on scheduler ordering or `runtime.Gosched()`?
- Are shared maps, slices, caches, and counters protected or partitioned by ownership?
- Can cancellation stop queued and in-flight work within an acceptable bound?
- Has goroutine-heavy code been checked with `go test -race` when feasible?
