package publicid

import (
	"errors"
	"strings"
	"testing"
)

func TestCodecRoundTripWithActiveVersionAndSecret(t *testing.T) {
	codec, err := NewCodec(Config{
		Prefix:        "n",
		ActiveVersion: 1,
		Secrets:       map[uint8]string{1: "notification-public-id-secret-v1"},
	})
	if err != nil {
		t.Fatalf("new codec: %v", err)
	}

	encoded, err := codec.Encode(2447119)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	if !strings.HasPrefix(encoded, "n1") {
		t.Fatalf("encoded id = %q, want n1 prefix", encoded)
	}
	decoded, err := codec.Decode(encoded)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if decoded != 2447119 {
		t.Fatalf("decoded id = %d, want 2447119", decoded)
	}
}

func TestCodecDecodesOldSecretVersion(t *testing.T) {
	oldCodec, err := NewCodec(Config{
		Prefix:        "n",
		ActiveVersion: 1,
		Secrets:       map[uint8]string{1: "notification-public-id-secret-v1"},
	})
	if err != nil {
		t.Fatalf("new old codec: %v", err)
	}
	oldID, err := oldCodec.Encode(42)
	if err != nil {
		t.Fatalf("encode old id: %v", err)
	}

	rotatedCodec, err := NewCodec(Config{
		Prefix:        "n",
		ActiveVersion: 2,
		Secrets: map[uint8]string{
			1: "notification-public-id-secret-v1",
			2: "notification-public-id-secret-v2",
		},
	})
	if err != nil {
		t.Fatalf("new rotated codec: %v", err)
	}

	decoded, err := rotatedCodec.Decode(oldID)
	if err != nil {
		t.Fatalf("decode old id after rotation: %v", err)
	}
	if decoded != 42 {
		t.Fatalf("decoded old id = %d, want 42", decoded)
	}
	newID, err := rotatedCodec.Encode(42)
	if err != nil {
		t.Fatalf("encode new id: %v", err)
	}
	if !strings.HasPrefix(newID, "n2") {
		t.Fatalf("new id = %q, want active version n2", newID)
	}
	if newID == oldID {
		t.Fatalf("rotated id should change when active secret changes")
	}
}

func TestCodecRejectsInvalidInputByClass(t *testing.T) {
	codec, err := NewCodec(Config{
		Prefix:        "n",
		ActiveVersion: 1,
		Secrets:       map[uint8]string{1: "notification-public-id-secret-v1"},
	})
	if err != nil {
		t.Fatalf("new codec: %v", err)
	}

	tests := []struct {
		name string
		id   string
		want error
	}{
		{name: "empty", id: "", want: ErrInvalidFormat},
		{name: "wrong prefix", id: "x1abcdef", want: ErrInvalidPrefix},
		{name: "unknown version", id: "n9abcdef", want: ErrUnknownVersion},
		{name: "bad alphabet", id: "n10OIl", want: ErrInvalidFormat},
		{name: "tampered checksum", id: "n1abc123", want: ErrInvalidChecksum},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := codec.Decode(tt.id)
			if !errors.Is(err, tt.want) {
				t.Fatalf("decode error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestCodecProducesUniqueIDsWithoutLeakingInternalSequence(t *testing.T) {
	codec, err := NewCodec(Config{
		Prefix:        "n",
		ActiveVersion: 1,
		Secrets:       map[uint8]string{1: "notification-public-id-secret-v1"},
	})
	if err != nil {
		t.Fatalf("new codec: %v", err)
	}

	seen := map[string]struct{}{}
	for id := uint64(1000); id < 1500; id++ {
		encoded, err := codec.Encode(id)
		if err != nil {
			t.Fatalf("encode %d: %v", id, err)
		}
		if _, ok := seen[encoded]; ok {
			t.Fatalf("duplicate encoded id %q", encoded)
		}
		seen[encoded] = struct{}{}
		if encoded == "n1"+base58Encode(id) {
			t.Fatalf("encoded id %q is direct base58 of internal id %d", encoded, id)
		}
		if len(encoded) > MaxLength {
			t.Fatalf("encoded id length = %d, want <= %d", len(encoded), MaxLength)
		}
	}
}
