package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	envPostgresDSN                    = "ZHICORE_NOTIFICATION_POSTGRES_DSN"
	envRedisAddr                      = "ZHICORE_NOTIFICATION_REDIS_ADDR"
	envRedisPassword                  = "ZHICORE_NOTIFICATION_REDIS_PASSWORD"
	envRedisDB                        = "ZHICORE_NOTIFICATION_REDIS_DB"
	envRabbitMQURL                    = "ZHICORE_NOTIFICATION_RABBITMQ_URL"
	envUserServiceBaseURL             = "ZHICORE_NOTIFICATION_USER_SERVICE_BASE_URL"
	envUserServiceTimeout             = "ZHICORE_NOTIFICATION_USER_SERVICE_TIMEOUT"
	envPublicIDActiveVersion          = "ZHICORE_NOTIFICATION_PUBLIC_ID_ACTIVE_VERSION"
	envPublicIDSecrets                = "ZHICORE_NOTIFICATION_PUBLIC_ID_SECRETS"
	envConsumedEventsRetention        = "ZHICORE_NOTIFICATION_CONSUMER_CONSUMED_EVENTS_RETENTION"
	envRealtimeFanoutTimeout          = "ZHICORE_NOTIFICATION_REALTIME_FANOUT_TIMEOUT"
	envCampaignClaimTimeout           = "ZHICORE_NOTIFICATION_CAMPAIGN_CLAIM_TIMEOUT"
	envCampaignShardBatchSize         = "ZHICORE_NOTIFICATION_CAMPAIGN_SHARD_BATCH_SIZE"
	envCampaignMaxConcurrentShardJobs = "ZHICORE_NOTIFICATION_CAMPAIGN_MAX_CONCURRENT_SHARD_JOBS"
	envHTTPAddr                       = "ZHICORE_NOTIFICATION_HTTP_ADDR"
	envHTTPReadHeaderTimeout          = "ZHICORE_NOTIFICATION_HTTP_READ_HEADER_TIMEOUT"
	envHTTPReadTimeout                = "ZHICORE_NOTIFICATION_HTTP_READ_TIMEOUT"
	envHTTPWriteTimeout               = "ZHICORE_NOTIFICATION_HTTP_WRITE_TIMEOUT"
	envHTTPIdleTimeout                = "ZHICORE_NOTIFICATION_HTTP_IDLE_TIMEOUT"
	envHTTPShutdownTimeout            = "ZHICORE_NOTIFICATION_HTTP_SHUTDOWN_TIMEOUT"
)

func LoadNotificationServerConfig(lookup func(string) (string, bool)) (NotificationServerConfig, error) {
	cfg := DefaultNotificationServerConfig()
	if lookup == nil {
		lookup = func(string) (string, bool) { return "", false }
	}

	requiredSeen := map[string]bool{}
	if err := overlayRequiredString(&cfg.Postgres.DSN, lookup, envPostgresDSN, requiredSeen); err != nil {
		return NotificationServerConfig{}, err
	}
	if err := overlayRequiredString(&cfg.Redis.Addr, lookup, envRedisAddr, requiredSeen); err != nil {
		return NotificationServerConfig{}, err
	}
	if err := overlayRequiredString(&cfg.RabbitMQ.URL, lookup, envRabbitMQURL, requiredSeen); err != nil {
		return NotificationServerConfig{}, err
	}
	if err := overlayRequiredString(&cfg.UserService.BaseURL, lookup, envUserServiceBaseURL, requiredSeen); err != nil {
		return NotificationServerConfig{}, err
	}
	if err := overlayPublicIDConfig(&cfg, lookup, requiredSeen); err != nil {
		return NotificationServerConfig{}, err
	}
	if err := overlayRuntimePolicyConfig(&cfg, lookup, requiredSeen); err != nil {
		return NotificationServerConfig{}, err
	}
	if err := overlayOptionalConfig(&cfg, lookup); err != nil {
		return NotificationServerConfig{}, err
	}
	if err := validateNotificationServerConfig(cfg, requiredSeen); err != nil {
		return NotificationServerConfig{}, err
	}

	return cfg, nil
}

func overlayRequiredString(target *string, lookup func(string) (string, bool), name string, seen map[string]bool) error {
	value, ok, err := lookupRequiredEnv(lookup, name)
	if err != nil {
		return err
	}
	if ok {
		*target = value
		seen[name] = true
	}
	return nil
}

func overlayPublicIDConfig(cfg *NotificationServerConfig, lookup func(string) (string, bool), seen map[string]bool) error {
	if value, ok, err := lookupRequiredEnv(lookup, envPublicIDActiveVersion); err != nil {
		return err
	} else if ok {
		parsed, err := parseUint8Env(envPublicIDActiveVersion, value)
		if err != nil {
			return err
		}
		cfg.PublicID.ActiveVersion = parsed
		seen[envPublicIDActiveVersion] = true
	}
	if value, ok, err := lookupRequiredEnv(lookup, envPublicIDSecrets); err != nil {
		return err
	} else if ok {
		secrets, err := parsePublicIDSecretsEnv(envPublicIDSecrets, value)
		if err != nil {
			return err
		}
		cfg.PublicID.Secrets = secrets
		seen[envPublicIDSecrets] = true
	}
	return nil
}

func overlayRuntimePolicyConfig(cfg *NotificationServerConfig, lookup func(string) (string, bool), seen map[string]bool) error {
	if value, ok, err := lookupRequiredEnv(lookup, envConsumedEventsRetention); err != nil {
		return err
	} else if ok {
		parsed, err := parseDurationEnv(envConsumedEventsRetention, value)
		if err != nil {
			return err
		}
		cfg.Consumer.ConsumedEventsRetention = parsed
		seen[envConsumedEventsRetention] = true
	}
	if value, ok, err := lookupRequiredEnv(lookup, envRealtimeFanoutTimeout); err != nil {
		return err
	} else if ok {
		parsed, err := parseDurationEnv(envRealtimeFanoutTimeout, value)
		if err != nil {
			return err
		}
		cfg.RealtimeFanout.Timeout = parsed
		seen[envRealtimeFanoutTimeout] = true
	}
	if value, ok, err := lookupRequiredEnv(lookup, envCampaignClaimTimeout); err != nil {
		return err
	} else if ok {
		parsed, err := parseDurationEnv(envCampaignClaimTimeout, value)
		if err != nil {
			return err
		}
		cfg.Campaign.ClaimTimeout = parsed
		seen[envCampaignClaimTimeout] = true
	}
	if value, ok, err := lookupRequiredEnv(lookup, envCampaignShardBatchSize); err != nil {
		return err
	} else if ok {
		parsed, err := parsePositiveIntEnv(envCampaignShardBatchSize, value)
		if err != nil {
			return err
		}
		cfg.Campaign.ShardBatchSize = parsed
		seen[envCampaignShardBatchSize] = true
	}
	if value, ok, err := lookupRequiredEnv(lookup, envCampaignMaxConcurrentShardJobs); err != nil {
		return err
	} else if ok {
		parsed, err := parsePositiveIntEnv(envCampaignMaxConcurrentShardJobs, value)
		if err != nil {
			return err
		}
		cfg.Campaign.MaxConcurrentShardJobs = parsed
		seen[envCampaignMaxConcurrentShardJobs] = true
	}
	return nil
}

func overlayOptionalConfig(cfg *NotificationServerConfig, lookup func(string) (string, bool)) error {
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
	if value, ok, err := lookupOptionalEnv(lookup, envHTTPAddr); err != nil {
		return err
	} else if ok {
		cfg.HTTP.Addr = value
	}
	if err := overlayOptionalDuration(&cfg.HTTP.ReadHeaderTimeout, lookup, envHTTPReadHeaderTimeout); err != nil {
		return err
	}
	if err := overlayOptionalDuration(&cfg.HTTP.ReadTimeout, lookup, envHTTPReadTimeout); err != nil {
		return err
	}
	if err := overlayOptionalDuration(&cfg.HTTP.WriteTimeout, lookup, envHTTPWriteTimeout); err != nil {
		return err
	}
	if err := overlayOptionalDuration(&cfg.HTTP.IdleTimeout, lookup, envHTTPIdleTimeout); err != nil {
		return err
	}
	if err := overlayOptionalDuration(&cfg.UserService.Timeout, lookup, envUserServiceTimeout); err != nil {
		return err
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
	return nil
}

func overlayOptionalDuration(target *time.Duration, lookup func(string) (string, bool), name string) error {
	value, ok, err := lookupOptionalEnv(lookup, name)
	if err != nil || !ok {
		return err
	}
	parsed, err := parseDurationEnv(name, value)
	if err != nil {
		return err
	}
	*target = parsed
	return nil
}

func parsePublicIDSecretsEnv(name, raw string) (map[uint8]string, error) {
	secrets := map[uint8]string{}
	for _, part := range strings.Split(raw, ",") {
		keyValue := strings.SplitN(strings.TrimSpace(part), ":", 2)
		if len(keyValue) != 2 {
			return nil, fmt.Errorf("%s: secret entry must use version:secret", name)
		}
		version, err := parseUint8Env(name, keyValue[0])
		if err != nil {
			return nil, err
		}
		secret := strings.TrimSpace(keyValue[1])
		if secret == "" {
			return nil, fmt.Errorf("%s: secret for version %d must not be empty", name, version)
		}
		secrets[version] = secret
	}
	return secrets, nil
}

func parseUint8Env(name, raw string) (uint8, error) {
	value, err := strconv.ParseUint(strings.TrimSpace(raw), 10, 8)
	if err != nil {
		return 0, fmt.Errorf("%s: parse uint8: %w", name, err)
	}
	if value == 0 || value > 9 {
		return 0, fmt.Errorf("%s: value must be 1-9", name)
	}
	return uint8(value), nil
}
