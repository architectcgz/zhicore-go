package preference

import (
	"fmt"
	"strings"
	"time"
)

const (
	ChannelInApp     = "IN_APP"
	ChannelWebsocket = "WEBSOCKET"
	ChannelEmail     = "EMAIL"
	ChannelSMS       = "SMS"

	SubscriptionAll        = "ALL"
	SubscriptionDigestOnly = "DIGEST_ONLY"
	SubscriptionMuted      = "MUTED"
)

type ChannelPreference struct {
	InApp     bool
	Websocket bool
	Email     bool
	SMS       bool
}

func NormalizeChannelPreference(input ChannelPreference) (ChannelPreference, error) {
	if input.SMS {
		return ChannelPreference{}, fmt.Errorf("sms channel is not enabled in this phase")
	}
	return input, nil
}

type DNDWindow struct {
	Enabled    bool
	StartTime  string
	EndTime    string
	Timezone   string
	Categories []string
	Channels   []string
}

func NormalizeDNDWindow(input DNDWindow) (DNDWindow, error) {
	input.StartTime = strings.TrimSpace(input.StartTime)
	input.EndTime = strings.TrimSpace(input.EndTime)
	input.Timezone = strings.TrimSpace(input.Timezone)
	if input.Timezone == "" {
		input.Timezone = "UTC"
	}
	if _, err := time.LoadLocation(input.Timezone); err != nil {
		return DNDWindow{}, fmt.Errorf("invalid dnd timezone")
	}
	if _, err := parseClock(input.StartTime); err != nil {
		return DNDWindow{}, err
	}
	if _, err := parseClock(input.EndTime); err != nil {
		return DNDWindow{}, err
	}
	// Cross-day DND windows such as 22:00-07:00 are valid; an equal boundary would
	// either mean a no-op or a full-day mute, so the contract rejects it explicitly.
	if input.StartTime == input.EndTime {
		return DNDWindow{}, fmt.Errorf("dnd start time must differ from end time")
	}
	input.Categories = normalizeStringSet(input.Categories)
	input.Channels = normalizeStringSet(input.Channels)
	return input, nil
}

func parseClock(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, fmt.Errorf("dnd time is required")
	}
	parsed, err := time.Parse("15:04", value)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid dnd time")
	}
	return parsed, nil
}

type AuthorSubscription struct {
	Level            string
	InAppEnabled     bool
	WebsocketEnabled bool
	EmailEnabled     bool
	DigestEnabled    bool
}

func NormalizeAuthorSubscription(input AuthorSubscription) (AuthorSubscription, error) {
	input.Level = strings.ToUpper(strings.TrimSpace(input.Level))
	if input.Level == "" {
		input.Level = SubscriptionAll
	}
	switch input.Level {
	case SubscriptionAll:
		return input, nil
	case SubscriptionDigestOnly:
		input.InAppEnabled = false
		input.WebsocketEnabled = false
		input.EmailEnabled = false
		input.DigestEnabled = true
		return input, nil
	case SubscriptionMuted:
		input.InAppEnabled = false
		input.WebsocketEnabled = false
		input.EmailEnabled = false
		input.DigestEnabled = false
		return input, nil
	default:
		return AuthorSubscription{}, fmt.Errorf("invalid author subscription level")
	}
}

func normalizeStringSet(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		normalized := strings.ToUpper(strings.TrimSpace(value))
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}
	return out
}
