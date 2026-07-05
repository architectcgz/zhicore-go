package file

import (
	"encoding/json"
	"testing"
)

func TestInternalHTTPContractPaths(t *testing.T) {
	if ValidateRefsPath == "" || ValidateRefsPath[0] != '/' {
		t.Fatalf("ValidateRefsPath = %q, want absolute path", ValidateRefsPath)
	}
}

func TestValidateRefsRequestJSONShape(t *testing.T) {
	payload, err := json.Marshal(ValidateRefsRequest{
		Usage: UsageContentBodyMedia,
		Refs:  []FileRef{{FileID: "file_1", Kind: "image"}},
	})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	want := `{"refs":[{"fileId":"file_1","kind":"image"}],"usage":"CONTENT_BODY_MEDIA"}`
	if string(payload) != want {
		t.Fatalf("payload = %s", payload)
	}
}

func TestValidateRefsResponseJSONShape(t *testing.T) {
	payload, err := json.Marshal(ValidateRefsResponse{InvalidFileIDs: []string{"file_missing"}})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if string(payload) != `{"invalidFileIds":["file_missing"]}` {
		t.Fatalf("payload = %s", payload)
	}
}
