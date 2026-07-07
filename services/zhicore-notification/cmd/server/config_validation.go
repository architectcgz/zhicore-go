package main

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"
)

func lookupRequiredEnv(lookup func(string) (string, bool), name string) (string, bool, error) {
	value, ok := lookup(name)
	if !ok {
		return "", false, nil
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return "", true, fmt.Errorf("%s: value must not be empty", name)
	}
	return value, true, nil
}

func lookupOptionalEnv(lookup func(string) (string, bool), name string) (string, bool, error) {
	value, ok := lookup(name)
	if !ok {
		return "", false, nil
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return "", false, fmt.Errorf("%s: value must not be empty", name)
	}
	return value, true, nil
}

func validateNotificationServerConfig(cfg NotificationServerConfig, requiredSeen map[string]bool) error {
	missing := missingRequiredEnv(cfg, requiredSeen)
	if len(missing) > 0 {
		return fmt.Errorf("missing required environment variables: %s", joinNames(missing))
	}
	if err := validateURLWithSchemes(envPostgresDSN, cfg.Postgres.DSN, "postgres", "postgresql"); err != nil {
		return err
	}
	if err := validateHostPort(envRedisAddr, cfg.Redis.Addr); err != nil {
		return err
	}
	if err := validateURLWithSchemes(envRabbitMQURL, cfg.RabbitMQ.URL, "amqp", "amqps"); err != nil {
		return err
	}
	if err := validateURLWithSchemes(envUserServiceBaseURL, cfg.UserService.BaseURL, "http", "https"); err != nil {
		return err
	}
	if cfg.UserService.Timeout <= 0 {
		return fmt.Errorf("%s: duration must be greater than zero", envUserServiceTimeout)
	}
	if strings.TrimSpace(cfg.PublicID.Secrets[cfg.PublicID.ActiveVersion]) == "" {
		return fmt.Errorf("%s: active version secret is required", envPublicIDSecrets)
	}
	if cfg.Consumer.ConsumedEventsRetention <= 0 {
		return fmt.Errorf("%s: duration must be greater than zero", envConsumedEventsRetention)
	}
	if cfg.RealtimeFanout.Timeout <= 0 {
		return fmt.Errorf("%s: duration must be greater than zero", envRealtimeFanoutTimeout)
	}
	if cfg.Campaign.ClaimTimeout <= 0 {
		return fmt.Errorf("%s: duration must be greater than zero", envCampaignClaimTimeout)
	}
	if cfg.Campaign.ShardBatchSize <= 0 || cfg.Campaign.ShardBatchSize > 10000 {
		return fmt.Errorf("%s: value must be between 1 and 10000", envCampaignShardBatchSize)
	}
	if cfg.Campaign.MaxConcurrentShardJobs <= 0 || cfg.Campaign.MaxConcurrentShardJobs > 128 {
		return fmt.Errorf("%s: value must be between 1 and 128", envCampaignMaxConcurrentShardJobs)
	}
	return nil
}

func missingRequiredEnv(cfg NotificationServerConfig, seen map[string]bool) []string {
	names := []string{
		envPostgresDSN,
		envRedisAddr,
		envRabbitMQURL,
		envUserServiceBaseURL,
		envPublicIDActiveVersion,
		envPublicIDSecrets,
		envConsumedEventsRetention,
		envRealtimeFanoutTimeout,
		envCampaignClaimTimeout,
		envCampaignShardBatchSize,
		envCampaignMaxConcurrentShardJobs,
	}
	missing := make([]string, 0, len(names))
	for _, name := range names {
		if !seen[name] {
			missing = append(missing, name)
		}
	}
	if cfg.Postgres.DSN == "" && !containsString(missing, envPostgresDSN) {
		missing = append(missing, envPostgresDSN)
	}
	if cfg.Redis.Addr == "" && !containsString(missing, envRedisAddr) {
		missing = append(missing, envRedisAddr)
	}
	if cfg.RabbitMQ.URL == "" && !containsString(missing, envRabbitMQURL) {
		missing = append(missing, envRabbitMQURL)
	}
	if cfg.UserService.BaseURL == "" && !containsString(missing, envUserServiceBaseURL) {
		missing = append(missing, envUserServiceBaseURL)
	}
	return missing
}

func parseDurationEnv(name, raw string) (time.Duration, error) {
	value, err := time.ParseDuration(strings.TrimSpace(raw))
	if err != nil {
		return 0, fmt.Errorf("%s: parse duration: %w", name, err)
	}
	if value <= 0 {
		return 0, fmt.Errorf("%s: duration must be greater than zero", name)
	}
	return value, nil
}

func parseShutdownTimeoutEnv(name, raw string) (time.Duration, error) {
	value, err := parseDurationEnv(name, raw)
	if err != nil {
		return 0, err
	}
	if value > 30*time.Second {
		return 0, fmt.Errorf("%s: duration must be less than or equal to 30s", name)
	}
	return value, nil
}

func parsePositiveIntEnv(name, raw string) (int, error) {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return 0, fmt.Errorf("%s: parse int: %w", name, err)
	}
	if value <= 0 {
		return 0, fmt.Errorf("%s: value must be greater than zero", name)
	}
	return value, nil
}

func parseNonNegativeIntEnv(name, raw string) (int, error) {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return 0, fmt.Errorf("%s: parse int: %w", name, err)
	}
	if value < 0 {
		return 0, fmt.Errorf("%s: value must be greater than or equal to zero", name)
	}
	return value, nil
}

func validateURLWithSchemes(name, raw string, allowedSchemes ...string) error {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return fmt.Errorf("%s: must be a valid %s URL", name, joinSchemes(allowedSchemes))
	}
	scheme := strings.ToLower(parsed.Scheme)
	if !containsString(allowedSchemes, scheme) {
		return fmt.Errorf("%s: must use %s", name, joinSchemes(allowedSchemes))
	}
	if parsed.Hostname() == "" {
		return fmt.Errorf("%s: hostname is required", name)
	}
	return nil
}

func validateHostPort(name, raw string) error {
	host, port, err := net.SplitHostPort(strings.TrimSpace(raw))
	if err != nil {
		return fmt.Errorf("%s: must be host:port", name)
	}
	if strings.TrimSpace(host) == "" || strings.TrimSpace(port) == "" {
		return fmt.Errorf("%s: host and port are required", name)
	}
	return nil
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func joinSchemes(schemes []string) string {
	if len(schemes) == 1 {
		return schemes[0]
	}
	return strings.Join(schemes[:len(schemes)-1], " or ") + " or " + schemes[len(schemes)-1]
}

func joinNames(names []string) string {
	return strings.Join(names, ", ")
}
