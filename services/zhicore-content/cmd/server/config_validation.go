package main

import (
	"fmt"
	"math"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
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

func missingRequiredEnv(cfg ContentServerConfig) []string {
	missing := make([]string, 0, 5)
	if cfg.Postgres.DSN == "" {
		missing = append(missing, envPostgresDSN)
	}
	if cfg.Mongo.URI == "" {
		missing = append(missing, envMongoURI)
	}
	if cfg.Redis.Addr == "" {
		missing = append(missing, envRedisAddr)
	}
	if cfg.RabbitMQ.URL == "" {
		missing = append(missing, envRabbitMQURL)
	}
	if cfg.UserService.BaseURL == "" {
		missing = append(missing, envUserServiceBaseURL)
	}
	if cfg.FileService.BaseURL == "" {
		missing = append(missing, envFileServiceBaseURL)
	}
	return missing
}

func validateContentServerConfig(cfg ContentServerConfig) error {
	missing := missingRequiredEnv(cfg)
	if len(missing) > 0 {
		return fmt.Errorf("missing required environment variables: %s", joinNames(missing))
	}
	if err := validateURLWithSchemes(envMongoURI, cfg.Mongo.URI, "mongodb", "mongodb+srv"); err != nil {
		return err
	}
	if err := validateURLWithSchemes(envRabbitMQURL, cfg.RabbitMQ.URL, "amqp", "amqps"); err != nil {
		return err
	}
	if cfg.Redis.DialTimeout <= 0 {
		return fmt.Errorf("%s: duration must be greater than zero", envRedisDialTimeout)
	}
	if cfg.Redis.ReadTimeout <= 0 {
		return fmt.Errorf("%s: duration must be greater than zero", envRedisReadTimeout)
	}
	if cfg.Redis.WriteTimeout <= 0 {
		return fmt.Errorf("%s: duration must be greater than zero", envRedisWriteTimeout)
	}
	if cfg.Redis.PoolSize <= 0 {
		return fmt.Errorf("%s: value must be greater than zero", envRedisPoolSize)
	}
	if err := validateURLWithSchemes(envUserServiceBaseURL, cfg.UserService.BaseURL, "http", "https"); err != nil {
		return err
	}
	if err := validateURLWithSchemes(envFileServiceBaseURL, cfg.FileService.BaseURL, "http", "https"); err != nil {
		return err
	}
	return nil
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

func parseRateLimitFallbackEnv(name, raw string) (ports.RateLimitFallback, error) {
	switch ports.RateLimitFallback(strings.TrimSpace(raw)) {
	case ports.RateLimitFallbackNone:
		return ports.RateLimitFallbackNone, nil
	case ports.RateLimitFallbackLocalMemory:
		return ports.RateLimitFallbackLocalMemory, nil
	case ports.RateLimitFallbackGatewayOnly:
		return ports.RateLimitFallbackGatewayOnly, nil
	default:
		return "", fmt.Errorf("%s: value must be none, local_memory, or gateway_only", name)
	}
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

func parseBoolEnv(name, raw string) (bool, error) {
	switch strings.TrimSpace(raw) {
	case "true":
		return true, nil
	case "false":
		return false, nil
	default:
		return false, fmt.Errorf("%s: value must be true or false", name)
	}
}

func parseByteSizeEnv(name, raw string) (int64, error) {
	value, err := parseByteSize(strings.TrimSpace(raw))
	if err != nil {
		return 0, fmt.Errorf("%s: parse size: %w", name, err)
	}
	return value, nil
}

func parseByteSize(raw string) (int64, error) {
	if raw == "" {
		return 0, fmt.Errorf("empty size")
	}

	units := []struct {
		suffix     string
		multiplier int64
	}{
		{suffix: "GIB", multiplier: 1 << 30},
		{suffix: "MIB", multiplier: 1 << 20},
		{suffix: "KIB", multiplier: 1 << 10},
		{suffix: "GB", multiplier: 1000 * 1000 * 1000},
		{suffix: "MB", multiplier: 1000 * 1000},
		{suffix: "KB", multiplier: 1000},
		{suffix: "B", multiplier: 1},
	}

	upper := strings.ToUpper(raw)
	for _, unit := range units {
		if !strings.HasSuffix(upper, unit.suffix) {
			continue
		}
		number := strings.TrimSpace(raw[:len(raw)-len(unit.suffix)])
		if number == "" {
			break
		}
		value, err := strconv.ParseInt(number, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid size %q", raw)
		}
		if value <= 0 {
			return 0, fmt.Errorf("size must be greater than zero")
		}
		if value > math.MaxInt64/unit.multiplier {
			return 0, fmt.Errorf("size overflows int64")
		}
		return value * unit.multiplier, nil
	}

	return 0, fmt.Errorf("size must include an explicit unit")
}

func validateURLWithSchemes(name, raw string, allowedSchemes ...string) error {
	parsed, err := url.Parse(raw)
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
