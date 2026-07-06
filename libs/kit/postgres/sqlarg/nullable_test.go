package sqlarg

import (
	"testing"
	"time"
)

func TestNullableSQLArgs(t *testing.T) {
	t.Run("string keeps whitespace but nulls empty", func(t *testing.T) {
		if got := String(""); got != nil {
			t.Fatalf("String(empty) = %#v, want nil", got)
		}
		if got := String("  "); got != "  " {
			t.Fatalf("String(spaces) = %#v, want original spaces", got)
		}
	})

	t.Run("non blank string nulls whitespace", func(t *testing.T) {
		if got := NonBlankString("  "); got != nil {
			t.Fatalf("NonBlankString(spaces) = %#v, want nil", got)
		}
		if got := NonBlankString(" value "); got != " value " {
			t.Fatalf("NonBlankString(value) = %#v, want original value", got)
		}
	})

	t.Run("string pointer distinguishes nil from explicit empty", func(t *testing.T) {
		if got := StringPtr(nil); got != nil {
			t.Fatalf("StringPtr(nil) = %#v, want nil", got)
		}
		value := "text"
		if got := StringPtr(&value); got != "text" {
			t.Fatalf("StringPtr(text) = %#v, want text", got)
		}
	})

	t.Run("optional string uses set flag", func(t *testing.T) {
		if got := OptionalString(false, "ignored"); got != nil {
			t.Fatalf("OptionalString(false) = %#v, want nil", got)
		}
		if got := OptionalString(true, ""); got != "" {
			t.Fatalf("OptionalString(true, empty) = %#v, want explicit empty", got)
		}
	})

	t.Run("numbers and time use zero as null", func(t *testing.T) {
		if got := Int(0); got != nil {
			t.Fatalf("Int(0) = %#v, want nil", got)
		}
		if got := Int64(0); got != nil {
			t.Fatalf("Int64(0) = %#v, want nil", got)
		}
		if got := Time(time.Time{}); got != nil {
			t.Fatalf("Time(zero) = %#v, want nil", got)
		}
		now := time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC)
		if got := Time(now); got != now {
			t.Fatalf("Time(now) = %#v, want now", got)
		}
	})
}
