package http_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/architectcgz/zhicore-go/tests/testkit"
)

func TestContentPublishFlow(t *testing.T) {
	server := testkit.StartContentServer(t)
	defer server.Close()

	create := testkit.DoJSON[createPostData](t, server.Client, http.MethodPost, testkit.Path(server.BaseURL, "/api/v1/posts"), map[string]string{"X-User-Id": "42"}, map[string]any{
		"title": "System publish flow",
	})
	testkit.RequireNonEmpty(t, "postId", create.Data.PostID)
	if create.Data.PostVersion != 1 {
		t.Fatalf("created postVersion = %d, want 1", create.Data.PostVersion)
	}

	blocks := []map[string]any{{
		"type": "paragraph",
		"children": []map[string]string{{
			"type": "text",
			"text": "hello content publish flow",
		}},
	}}
	save := testkit.DoJSON[saveDraftBodyData](t, server.Client, http.MethodPut, testkit.Path(server.BaseURL, "/api/v1/posts/"+create.Data.PostID+"/draft/body"), map[string]string{"X-User-Id": "42"}, map[string]any{
		"basePostVersion":   create.Data.PostVersion,
		"baseDraftBodyId":   "",
		"baseDraftBodyHash": "",
		"schemaVersion":     1,
		"blocks":            blocks,
	})
	testkit.RequireNonEmpty(t, "draftBodyId", save.Data.DraftBodyID)
	testkit.RequireNonEmpty(t, "draftBodyHash", save.Data.DraftBodyHash)
	if save.Data.PostVersion <= create.Data.PostVersion {
		t.Fatalf("saved postVersion = %d, want greater than %d", save.Data.PostVersion, create.Data.PostVersion)
	}

	publish := testkit.DoJSON[publishPostData](t, server.Client, http.MethodPost, testkit.Path(server.BaseURL, "/api/v1/posts/"+create.Data.PostID+"/publish"), map[string]string{"X-User-Id": "42"}, map[string]any{
		"basePostVersion": save.Data.PostVersion,
		"draftBodyId":     save.Data.DraftBodyID,
		"draftBodyHash":   save.Data.DraftBodyHash,
	})
	if publish.Data.PostVersion <= save.Data.PostVersion {
		t.Fatalf("published postVersion = %d, want greater than %d", publish.Data.PostVersion, save.Data.PostVersion)
	}

	body := testkit.DoJSON[postBodyData](t, server.Client, http.MethodGet, testkit.Path(server.BaseURL, "/api/v1/posts/"+create.Data.PostID+"/body"), nil, nil)
	if body.Data.BodyID == "" || body.Data.BodyID == save.Data.DraftBodyID {
		t.Fatalf("published bodyId = %q, want snapshot distinct from draft %q", body.Data.BodyID, save.Data.DraftBodyID)
	}
	if body.Data.SchemaVersion != 1 || body.Data.Format != "blocks" {
		t.Fatalf("published body schema/format = %d/%q", body.Data.SchemaVersion, body.Data.Format)
	}
	if body.Data.ContentHash != save.Data.DraftBodyHash {
		t.Fatalf("published contentHash = %q, want draft hash %q", body.Data.ContentHash, save.Data.DraftBodyHash)
	}
	if len(body.Data.Blocks) != 1 {
		t.Fatalf("published blocks = %s, want one paragraph", string(body.Data.Blocks))
	}
}

type createPostData struct {
	PostID      string `json:"postId"`
	PostVersion int64  `json:"postVersion"`
}

type saveDraftBodyData struct {
	PostID        string `json:"postId"`
	PostVersion   int64  `json:"postVersion"`
	DraftBodyID   string `json:"draftBodyId"`
	DraftBodyHash string `json:"draftBodyHash"`
}

type publishPostData struct {
	PostID      string `json:"postId"`
	PostVersion int64  `json:"postVersion"`
	PublishedAt string `json:"publishedAt"`
}

type postBodyData struct {
	BodyID        string          `json:"bodyId"`
	SchemaVersion int             `json:"schemaVersion"`
	Format        string          `json:"format"`
	Blocks        json.RawMessage `json:"blocks"`
	PlainText     string          `json:"plainText"`
	ContentHash   string          `json:"contentHash"`
}
