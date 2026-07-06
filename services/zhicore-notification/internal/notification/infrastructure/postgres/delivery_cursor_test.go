package postgres

import (
	"testing"
	"time"
)

func TestDeliveryCursorRoundTrip(t *testing.T) {
	createdAt := time.Date(2026, 7, 6, 18, 0, 1, 234, time.UTC)
	cursor := encodeDeliveryCursor(createdAt, "d1abc")

	decodedAt, publicID, ok := decodeDeliveryCursor(cursor)
	if !ok {
		t.Fatal("decodeDeliveryCursor() ok = false, want true")
	}
	if !decodedAt.Equal(createdAt) || publicID != "d1abc" {
		t.Fatalf("decoded cursor = %s/%s, want %s/d1abc", decodedAt, publicID, createdAt)
	}
}

func TestDeliveryCursorRejectsInvalidCursor(t *testing.T) {
	if _, _, ok := decodeDeliveryCursor("not-base64"); ok {
		t.Fatal("decodeDeliveryCursor(invalid) ok = true, want false")
	}
}
