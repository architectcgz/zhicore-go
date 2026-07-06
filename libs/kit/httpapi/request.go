package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var errTrailingJSONValue = errors.New("request body must contain a single JSON value")

func DecodeJSONBody(r *http.Request, target any) error {
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return errTrailingJSONValue
		}
		return err
	}
	return nil
}

func DecodeJSONBodyLimited(w http.ResponseWriter, r *http.Request, maxBytes int64, target any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
	return DecodeJSONBody(r, target)
}

func ParsePositiveInt(raw string, defaultValue int, max int) (int, error) {
	if strings.TrimSpace(raw) == "" {
		return defaultValue, nil
	}
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return 0, err
	}
	if value <= 0 {
		return 0, errors.New("value must be positive")
	}
	if max > 0 && value > max {
		return 0, errors.New("value exceeds max")
	}
	return value, nil
}

func FormatRFC3339UTC(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}
