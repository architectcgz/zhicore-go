package sqlarg

import (
	"strings"
	"time"
)

func String(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func NonBlankString(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func StringPtr(value *string) any {
	if value == nil {
		return nil
	}
	return String(*value)
}

func OptionalString(set bool, value string) any {
	if !set {
		return nil
	}
	return value
}

func Int(value int) any {
	if value == 0 {
		return nil
	}
	return value
}

func Int64(value int64) any {
	if value == 0 {
		return nil
	}
	return value
}

func Time(value time.Time) any {
	if value.IsZero() {
		return nil
	}
	return value
}
