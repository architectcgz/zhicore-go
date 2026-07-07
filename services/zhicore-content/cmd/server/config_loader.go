package main

import (
	"strings"

	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
)

const (
	envPostgresDSN            = "ZHICORE_CONTENT_POSTGRES_DSN"
	envMongoURI               = "ZHICORE_CONTENT_MONGO_URI"
	envRedisAddr              = "ZHICORE_CONTENT_REDIS_ADDR"
	envRedisUsername          = "ZHICORE_CONTENT_REDIS_USERNAME"
	envRedisPassword          = "ZHICORE_CONTENT_REDIS_PASSWORD"
	envRedisDB                = "ZHICORE_CONTENT_REDIS_DB"
	envRedisDialTimeout       = "ZHICORE_CONTENT_REDIS_DIAL_TIMEOUT"
	envRedisReadTimeout       = "ZHICORE_CONTENT_REDIS_READ_TIMEOUT"
	envRedisWriteTimeout      = "ZHICORE_CONTENT_REDIS_WRITE_TIMEOUT"
	envRedisPoolSize          = "ZHICORE_CONTENT_REDIS_POOL_SIZE"
	envRabbitMQURL            = "ZHICORE_CONTENT_RABBITMQ_URL"
	envRabbitMQExchange       = "ZHICORE_CONTENT_RABBITMQ_EXCHANGE"
	envRabbitMQConfirmTimeout = "ZHICORE_CONTENT_RABBITMQ_PUBLISH_CONFIRM_TIMEOUT"
	envUserServiceBaseURL     = "ZHICORE_CONTENT_USER_SERVICE_BASE_URL"
	envFileServiceBaseURL     = "ZHICORE_CONTENT_FILE_SERVICE_BASE_URL"
	envHTTPAddr               = "ZHICORE_CONTENT_HTTP_ADDR"
	envHTTPReadHeaderTimeout  = "ZHICORE_CONTENT_HTTP_READ_HEADER_TIMEOUT"
	envHTTPReadTimeout        = "ZHICORE_CONTENT_HTTP_READ_TIMEOUT"
	envHTTPWriteTimeout       = "ZHICORE_CONTENT_HTTP_WRITE_TIMEOUT"
	envHTTPIdleTimeout        = "ZHICORE_CONTENT_HTTP_IDLE_TIMEOUT"
	envHTTPShutdownTimeout    = "ZHICORE_CONTENT_HTTP_SHUTDOWN_TIMEOUT"
	envHTTPMaxJSONBody        = "ZHICORE_CONTENT_HTTP_MAX_JSON_BODY"
	envCleanupEnabled         = "ZHICORE_CONTENT_WORKERS_CLEANUP_ENABLED"
	envRepairEnabled          = "ZHICORE_CONTENT_WORKERS_REPAIR_ENABLED"
	envOutboxEnabled          = "ZHICORE_CONTENT_WORKERS_OUTBOX_ENABLED"
)

func LoadContentServerConfig(lookup func(string) (string, bool)) (ContentServerConfig, error) {
	cfg := DefaultContentServerConfig()
	if lookup == nil {
		lookup = func(string) (string, bool) { return "", false }
	}

	if value, found, err := lookupRequiredEnv(lookup, envPostgresDSN); err != nil {
		return ContentServerConfig{}, err
	} else if found {
		cfg.Postgres.DSN = value
	}
	if value, found, err := lookupRequiredEnv(lookup, envMongoURI); err != nil {
		return ContentServerConfig{}, err
	} else if found {
		cfg.Mongo.URI = value
	}
	if value, found, err := lookupRequiredEnv(lookup, envRedisAddr); err != nil {
		return ContentServerConfig{}, err
	} else if found {
		cfg.Redis.Addr = value
	}
	if value, found, err := lookupRequiredEnv(lookup, envRabbitMQURL); err != nil {
		return ContentServerConfig{}, err
	} else if found {
		cfg.RabbitMQ.URL = value
	}
	if value, found, err := lookupRequiredEnv(lookup, envUserServiceBaseURL); err != nil {
		return ContentServerConfig{}, err
	} else if found {
		cfg.UserService.BaseURL = value
	}
	if value, found, err := lookupRequiredEnv(lookup, envFileServiceBaseURL); err != nil {
		return ContentServerConfig{}, err
	} else if found {
		cfg.FileService.BaseURL = value
	}
	if value, ok, err := lookupOptionalEnv(lookup, envHTTPAddr); err != nil {
		return ContentServerConfig{}, err
	} else if ok {
		cfg.HTTP.Addr = value
	}

	if err := overlayHTTPConfig(&cfg, lookup); err != nil {
		return ContentServerConfig{}, err
	}
	if err := overlayRedisConfig(&cfg, lookup); err != nil {
		return ContentServerConfig{}, err
	}
	if err := overlayRabbitMQConfig(&cfg, lookup); err != nil {
		return ContentServerConfig{}, err
	}
	if err := overlayWorkerConfig(&cfg, lookup); err != nil {
		return ContentServerConfig{}, err
	}
	if err := overlayRateLimitConfig(&cfg, lookup); err != nil {
		return ContentServerConfig{}, err
	}
	if err := validateContentServerConfig(cfg); err != nil {
		return ContentServerConfig{}, err
	}

	return cfg, nil
}

func overlayHTTPConfig(cfg *ContentServerConfig, lookup func(string) (string, bool)) error {
	if value, ok, err := lookupOptionalEnv(lookup, envHTTPReadHeaderTimeout); err != nil {
		return err
	} else if ok {
		parsed, err := parseDurationEnv(envHTTPReadHeaderTimeout, value)
		if err != nil {
			return err
		}
		cfg.HTTP.ReadHeaderTimeout = parsed
	}
	if value, ok, err := lookupOptionalEnv(lookup, envHTTPReadTimeout); err != nil {
		return err
	} else if ok {
		parsed, err := parseDurationEnv(envHTTPReadTimeout, value)
		if err != nil {
			return err
		}
		cfg.HTTP.ReadTimeout = parsed
	}
	if value, ok, err := lookupOptionalEnv(lookup, envHTTPWriteTimeout); err != nil {
		return err
	} else if ok {
		parsed, err := parseDurationEnv(envHTTPWriteTimeout, value)
		if err != nil {
			return err
		}
		cfg.HTTP.WriteTimeout = parsed
	}
	if value, ok, err := lookupOptionalEnv(lookup, envHTTPIdleTimeout); err != nil {
		return err
	} else if ok {
		parsed, err := parseDurationEnv(envHTTPIdleTimeout, value)
		if err != nil {
			return err
		}
		cfg.HTTP.IdleTimeout = parsed
	}
	if value, ok, err := lookupOptionalEnv(lookup, envHTTPShutdownTimeout); err != nil {
		return err
	} else if ok {
		parsed, err := parseShutdownTimeoutEnv(envHTTPShutdownTimeout, value)
		if err != nil {
			return err
		}
		cfg.HTTP.ShutdownTimeout = parsed
	}
	if value, ok, err := lookupOptionalEnv(lookup, envHTTPMaxJSONBody); err != nil {
		return err
	} else if ok {
		parsed, err := parseByteSizeEnv(envHTTPMaxJSONBody, value)
		if err != nil {
			return err
		}
		cfg.HTTP.MaxJSONBodyBytes = parsed
	}
	return nil
}

func overlayRedisConfig(cfg *ContentServerConfig, lookup func(string) (string, bool)) error {
	if value, ok, err := lookupOptionalEnv(lookup, envRedisUsername); err != nil {
		return err
	} else if ok {
		cfg.Redis.Username = value
	}
	if value, ok, err := lookupOptionalEnv(lookup, envRedisPassword); err != nil {
		return err
	} else if ok {
		cfg.Redis.Password = value
	}
	if value, ok, err := lookupOptionalEnv(lookup, envRedisDB); err != nil {
		return err
	} else if ok {
		parsed, err := parseNonNegativeIntEnv(envRedisDB, value)
		if err != nil {
			return err
		}
		cfg.Redis.DB = parsed
	}
	if value, ok, err := lookupOptionalEnv(lookup, envRedisDialTimeout); err != nil {
		return err
	} else if ok {
		parsed, err := parseDurationEnv(envRedisDialTimeout, value)
		if err != nil {
			return err
		}
		cfg.Redis.DialTimeout = parsed
	}
	if value, ok, err := lookupOptionalEnv(lookup, envRedisReadTimeout); err != nil {
		return err
	} else if ok {
		parsed, err := parseDurationEnv(envRedisReadTimeout, value)
		if err != nil {
			return err
		}
		cfg.Redis.ReadTimeout = parsed
	}
	if value, ok, err := lookupOptionalEnv(lookup, envRedisWriteTimeout); err != nil {
		return err
	} else if ok {
		parsed, err := parseDurationEnv(envRedisWriteTimeout, value)
		if err != nil {
			return err
		}
		cfg.Redis.WriteTimeout = parsed
	}
	if value, ok, err := lookupOptionalEnv(lookup, envRedisPoolSize); err != nil {
		return err
	} else if ok {
		parsed, err := parsePositiveIntEnv(envRedisPoolSize, value)
		if err != nil {
			return err
		}
		cfg.Redis.PoolSize = parsed
	}
	return nil
}

func overlayRabbitMQConfig(cfg *ContentServerConfig, lookup func(string) (string, bool)) error {
	if value, ok, err := lookupOptionalEnv(lookup, envRabbitMQExchange); err != nil {
		return err
	} else if ok {
		cfg.RabbitMQ.Exchange = value
	}
	if value, ok, err := lookupOptionalEnv(lookup, envRabbitMQConfirmTimeout); err != nil {
		return err
	} else if ok {
		parsed, err := parseDurationEnv(envRabbitMQConfirmTimeout, value)
		if err != nil {
			return err
		}
		cfg.RabbitMQ.PublishConfirmTimeout = parsed
	}
	return nil
}

func overlayRateLimitConfig(cfg *ContentServerConfig, lookup func(string) (string, bool)) error {
	for limitType := range cfg.RateLimit.Rules {
		rule := cfg.RateLimit.Rules[limitType]
		prefix := rateLimitEnvPrefix(limitType)
		if value, ok, err := lookupOptionalEnv(lookup, prefix+"_LIMIT"); err != nil {
			return err
		} else if ok {
			parsed, err := parsePositiveIntEnv(prefix+"_LIMIT", value)
			if err != nil {
				return err
			}
			rule.Limit = parsed
		}
		if value, ok, err := lookupOptionalEnv(lookup, prefix+"_WINDOW"); err != nil {
			return err
		} else if ok {
			parsed, err := parseDurationEnv(prefix+"_WINDOW", value)
			if err != nil {
				return err
			}
			rule.Window = parsed
		}
		if value, ok, err := lookupOptionalEnv(lookup, prefix+"_FALLBACK"); err != nil {
			return err
		} else if ok {
			parsed, err := parseRateLimitFallbackEnv(prefix+"_FALLBACK", value)
			if err != nil {
				return err
			}
			rule.Fallback = parsed
		}
		if value, ok, err := lookupOptionalEnv(lookup, prefix+"_FAIL_CLOSED"); err != nil {
			return err
		} else if ok {
			parsed, err := parseBoolEnv(prefix+"_FAIL_CLOSED", value)
			if err != nil {
				return err
			}
			rule.FailClosed = parsed
		}
		cfg.RateLimit.Rules[limitType] = rule
	}
	return nil
}

func rateLimitEnvPrefix(limitType ports.RateLimitType) string {
	return "ZHICORE_CONTENT_RATE_LIMIT_" + strings.ToUpper(string(limitType))
}

func overlayWorkerConfig(cfg *ContentServerConfig, lookup func(string) (string, bool)) error {
	if value, ok, err := lookupOptionalEnv(lookup, envCleanupEnabled); err != nil {
		return err
	} else if ok {
		parsed, err := parseBoolEnv(envCleanupEnabled, value)
		if err != nil {
			return err
		}
		cfg.Workers.CleanupEnabled = parsed
	}
	if value, ok, err := lookupOptionalEnv(lookup, envRepairEnabled); err != nil {
		return err
	} else if ok {
		parsed, err := parseBoolEnv(envRepairEnabled, value)
		if err != nil {
			return err
		}
		cfg.Workers.RepairEnabled = parsed
	}
	if value, ok, err := lookupOptionalEnv(lookup, envOutboxEnabled); err != nil {
		return err
	} else if ok {
		parsed, err := parseBoolEnv(envOutboxEnabled, value)
		if err != nil {
			return err
		}
		cfg.Workers.OutboxEnabled = parsed
	}
	return nil
}
