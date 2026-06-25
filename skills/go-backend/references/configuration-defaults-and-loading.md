# Configuration Defaults and Loading

Keep configuration shape, defaults, and loading separate. Configuration is an application boundary concern; handlers, use cases, repositories, clients, and workers should receive typed settings instead of reading environment variables or files directly.

## Rule

- Define typed config structs separately from loading logic.
- Define defaults in pure functions with no I/O.
- Load external configuration by overlaying environment, file, or flags onto defaults, then validate.
- Construct config at the process root and inject narrow config structs into use cases, clients, workers, and adapters.
- Do not use package-global mutable config, `init()` loading, or deep `os.Getenv` calls inside business code.
- Keep durations, retry budgets, backoff, queue limits, pool sizes, and dependency endpoints named and validated.

## Bad

```go
func (uc *CreateArticleUseCase) Execute(ctx context.Context, cmd CreateArticleCommand) (*Article, error) {
	timeout := 800 * time.Millisecond
	if raw := os.Getenv("CREATE_ARTICLE_TIMEOUT"); raw != "" {
		parsed, err := time.ParseDuration(raw)
		if err == nil {
			timeout = parsed
		}
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	return uc.repo.Create(ctx, articleFrom(cmd))
}
```

Problems:

- use case reads process environment directly
- default, load, parse, validation, and business execution are mixed
- invalid config silently falls back or fails late
- tests must mutate global process state to cover timeout behavior

## Better

`config.go`: shape and validation only.

```go
package appconfig

import (
	"fmt"
	"time"
)

type Config struct {
	HTTP     HTTPConfig
	Article  ArticleConfig
	Billing  BillingConfig
	Database DatabaseConfig
}

type HTTPConfig struct {
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

type ArticleConfig struct {
	CreateTimeout time.Duration
	CacheTimeout  time.Duration
}

type BillingConfig struct {
	Endpoint      string
	ChargeTimeout time.Duration
}

type DatabaseConfig struct {
	DSN             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

func (c Config) Validate() error {
	if err := c.HTTP.Validate(); err != nil {
		return fmt.Errorf("http config: %w", err)
	}
	if err := c.Article.Validate(); err != nil {
		return fmt.Errorf("article config: %w", err)
	}
	if err := c.Billing.Validate(); err != nil {
		return fmt.Errorf("billing config: %w", err)
	}
	if err := c.Database.Validate(); err != nil {
		return fmt.Errorf("database config: %w", err)
	}
	return nil
}

func (c ArticleConfig) Validate() error {
	if c.CreateTimeout <= 0 {
		return fmt.Errorf("create timeout must be positive")
	}
	if c.CacheTimeout <= 0 {
		return fmt.Errorf("cache timeout must be positive")
	}
	return nil
}

func (c HTTPConfig) Validate() error {
	if c.ReadTimeout <= 0 || c.WriteTimeout <= 0 || c.IdleTimeout <= 0 {
		return fmt.Errorf("timeouts must be positive")
	}
	return nil
}

func (c BillingConfig) Validate() error {
	if c.Endpoint == "" {
		return fmt.Errorf("endpoint is required")
	}
	if c.ChargeTimeout <= 0 {
		return fmt.Errorf("charge timeout must be positive")
	}
	return nil
}

func (c DatabaseConfig) Validate() error {
	if c.DSN == "" {
		return fmt.Errorf("dsn is required")
	}
	if c.MaxOpenConns <= 0 || c.MaxIdleConns < 0 {
		return fmt.Errorf("invalid connection pool limits")
	}
	if c.MaxIdleConns > c.MaxOpenConns {
		return fmt.Errorf("max idle conns cannot exceed max open conns")
	}
	if c.ConnMaxLifetime <= 0 {
		return fmt.Errorf("conn max lifetime must be positive")
	}
	return nil
}
```

`defaults.go`: defaults only, no environment or file reads.

```go
package appconfig

import "time"

func Default() Config {
	return Config{
		HTTP: HTTPConfig{
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
		Article: ArticleConfig{
			CreateTimeout: 800 * time.Millisecond,
			CacheTimeout:  30 * time.Second,
		},
		Billing: BillingConfig{
			Endpoint:      "https://billing.internal",
			ChargeTimeout: 2 * time.Second,
		},
		Database: DatabaseConfig{
			// Required secret-like values can intentionally have no default.
			// The loader must fill them before validation.
			MaxOpenConns:    20,
			MaxIdleConns:    10,
			ConnMaxLifetime: 30 * time.Minute,
		},
	}
}
```

`load_env.go`: loading and parsing only.

```go
package appconfig

import (
	"fmt"
	"strconv"
	"time"
)

type LookupEnv func(key string) (string, bool)

func LoadFromEnv(lookup LookupEnv) (Config, error) {
	cfg := Default()

	if v, ok := lookup("HTTP_READ_TIMEOUT"); ok {
		parsed, err := parseDuration("HTTP_READ_TIMEOUT", v)
		if err != nil {
			return Config{}, err
		}
		cfg.HTTP.ReadTimeout = parsed
	}
	if v, ok := lookup("ARTICLE_CREATE_TIMEOUT"); ok {
		parsed, err := parseDuration("ARTICLE_CREATE_TIMEOUT", v)
		if err != nil {
			return Config{}, err
		}
		cfg.Article.CreateTimeout = parsed
	}
	if v, ok := lookup("ARTICLE_CACHE_TIMEOUT"); ok {
		parsed, err := parseDuration("ARTICLE_CACHE_TIMEOUT", v)
		if err != nil {
			return Config{}, err
		}
		cfg.Article.CacheTimeout = parsed
	}
	if v, ok := lookup("BILLING_ENDPOINT"); ok {
		cfg.Billing.Endpoint = v
	}
	if v, ok := lookup("BILLING_CHARGE_TIMEOUT"); ok {
		parsed, err := parseDuration("BILLING_CHARGE_TIMEOUT", v)
		if err != nil {
			return Config{}, err
		}
		cfg.Billing.ChargeTimeout = parsed
	}
	if v, ok := lookup("DB_DSN"); ok {
		cfg.Database.DSN = v
	}
	if v, ok := lookup("DB_MAX_OPEN_CONNS"); ok {
		parsed, err := parseInt("DB_MAX_OPEN_CONNS", v)
		if err != nil {
			return Config{}, err
		}
		cfg.Database.MaxOpenConns = parsed
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func parseDuration(name, raw string) (time.Duration, error) {
	value, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("%s: parse duration: %w", name, err)
	}
	return value, nil
}

func parseInt(name, raw string) (int, error) {
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("%s: parse int: %w", name, err)
	}
	return value, nil
}
```

`wire.go`: construct at the process root and inject narrow config.

```go
func main() {
	cfg, err := appconfig.LoadFromEnv(os.LookupEnv)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	repo := postgres.NewArticleRepo(db)
	search := searchclient.New(...)
	outbox := postgres.NewOutbox(db)
	cache := redis.NewCache(...)

	createArticle := article.NewCreateUseCase(article.CreateUseCaseDeps{
		Repo:     repo,
		Search:   search,
		Outbox:   outbox,
		Cache:    cache,
		Timeouts: article.Timeouts{
			Create: cfg.Article.CreateTimeout,
			Cache:  cfg.Article.CacheTimeout,
		},
	})

	billing, err := billingclient.New(http.DefaultClient, billingclient.Config{
		Endpoint:      cfg.Billing.Endpoint,
		ChargeTimeout: cfg.Billing.ChargeTimeout,
	})
	if err != nil {
		log.Fatalf("create billing client: %v", err)
	}

	_ = createArticle
	_ = billing
}
```

## Use Case Consumption

Use cases consume typed config. They do not load it.

```go
package article

import "time"

type Timeouts struct {
	Create time.Duration
	Cache  time.Duration
}

type CreateUseCaseDeps struct {
	Repo     ArticleRepo
	Search   SearchIndex
	Outbox   Outbox
	Cache    Cache
	Timeouts Timeouts
}

type CreateUseCase struct {
	repo     ArticleRepo
	search   SearchIndex
	outbox   Outbox
	cache    Cache
	timeouts Timeouts
}

func NewCreateUseCase(deps CreateUseCaseDeps) *CreateUseCase {
	return &CreateUseCase{
		repo:     deps.Repo,
		search:   deps.Search,
		outbox:   deps.Outbox,
		cache:    deps.Cache,
		timeouts: deps.Timeouts,
	}
}
```

## Review Checklist

- Are defaults pure and separate from external loading?
- Are config structs and validation separate from env/file parsing?
- Is external config loaded once at the process root?
- Are narrow typed config structs injected into use cases, clients, workers, and adapters?
- Are invalid duration, endpoint, retry, backoff, queue, and pool settings rejected at startup?
- Are business methods free of `os.Getenv`, config file reads, package-global mutable config, and magic duration literals?
