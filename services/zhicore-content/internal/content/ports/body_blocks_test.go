package ports

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBlocksMarshalJSONPreservesNilBlocksAsNull(t *testing.T) {
	raw, err := json.Marshal(PostBodyWriteInput{SchemaVersion: 1})
	if err != nil {
		t.Fatalf("json.Marshal returned error: %v", err)
	}

	body := string(raw)
	if !strings.Contains(body, `"blocks":null`) {
		t.Fatalf("PostBodyWriteInput JSON = %s, want nil blocks encoded as null", body)
	}
	if strings.Contains(body, `"blocks":[]`) {
		t.Fatalf("PostBodyWriteInput JSON = %s, must not turn nil blocks into []", body)
	}
}

func TestBlocksMarshalJSONRejectsTypedNilBlock(t *testing.T) {
	var block *UnsupportedBlock

	_, err := json.Marshal(Blocks{block})

	if err == nil {
		t.Fatal("json.Marshal returned nil error, want typed nil block error")
	}
}
