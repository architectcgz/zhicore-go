package testkit

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	filecontract "github.com/architectcgz/zhicore-go/libs/contracts/clients/file"
	usercontract "github.com/architectcgz/zhicore-go/libs/contracts/clients/user"
	_ "github.com/lib/pq"
)

const (
	envSystemBaseURL     = "ZHICORE_CONTENT_SYSTEM_BASE_URL"
	envSystemPostgresDSN = "ZHICORE_CONTENT_SYSTEM_POSTGRES_DSN"
	envSystemMongoURI    = "ZHICORE_CONTENT_SYSTEM_MONGO_URI"
	envSystemRabbitMQURL = "ZHICORE_CONTENT_SYSTEM_RABBITMQ_URL"
)

type ContentServerFixture struct {
	BaseURL string
	Client  *http.Client
	cleanup func()
}

func (f ContentServerFixture) Close() {
	if f.cleanup != nil {
		f.cleanup()
	}
}

func StartContentServer(t *testing.T) ContentServerFixture {
	t.Helper()

	if baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv(envSystemBaseURL)), "/"); baseURL != "" {
		return ContentServerFixture{BaseURL: baseURL, Client: http.DefaultClient}
	}

	postgresDSN := strings.TrimSpace(os.Getenv(envSystemPostgresDSN))
	mongoURI := strings.TrimSpace(os.Getenv(envSystemMongoURI))
	rabbitURL := strings.TrimSpace(os.Getenv(envSystemRabbitMQURL))
	if postgresDSN == "" || mongoURI == "" || rabbitURL == "" {
		t.Skipf("set %s or all of %s, %s, %s to run Content HTTP system test with real dependencies",
			envSystemBaseURL, envSystemPostgresDSN, envSystemMongoURI, envSystemRabbitMQURL)
	}

	root := repoRoot(t)
	applyContentMigrations(t, root, postgresDSN)

	userServer := newFakeUserServer(t)
	fileServer := newFakeFileServer(t)
	t.Cleanup(userServer.Close)
	t.Cleanup(fileServer.Close)

	addr := reserveLocalAddr(t)
	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, "go", "run", "./services/zhicore-content/cmd/server")
	cmd.Dir = root
	cmd.Env = append(os.Environ(),
		"ZHICORE_CONTENT_HTTP_ADDR="+addr,
		"ZHICORE_CONTENT_POSTGRES_DSN="+postgresDSN,
		"ZHICORE_CONTENT_MONGO_URI="+mongoURI,
		"ZHICORE_CONTENT_RABBITMQ_URL="+rabbitURL,
		"ZHICORE_CONTENT_USER_SERVICE_BASE_URL="+userServer.URL,
		"ZHICORE_CONTENT_FILE_SERVICE_BASE_URL="+fileServer.URL,
		"ZHICORE_CONTENT_HTTP_READ_HEADER_TIMEOUT=2s",
		"ZHICORE_CONTENT_HTTP_READ_TIMEOUT=5s",
		"ZHICORE_CONTENT_HTTP_WRITE_TIMEOUT=10s",
		"ZHICORE_CONTENT_HTTP_IDLE_TIMEOUT=30s",
		"ZHICORE_CONTENT_HTTP_SHUTDOWN_TIMEOUT=5s",
		"ZHICORE_CONTENT_HTTP_MAX_JSON_BODY=1MiB",
		"ZHICORE_CONTENT_WORKERS_CLEANUP_ENABLED=false",
		"ZHICORE_CONTENT_WORKERS_REPAIR_ENABLED=false",
		"ZHICORE_CONTENT_WORKERS_OUTBOX_ENABLED=false",
	)

	output := &strings.Builder{}
	cmd.Stdout = output
	cmd.Stderr = output
	if err := cmd.Start(); err != nil {
		cancel()
		t.Fatalf("start content server: %v", err)
	}

	baseURL := "http://" + addr
	waitForReady(t, http.DefaultClient, baseURL, output)
	return ContentServerFixture{
		BaseURL: baseURL,
		Client:  http.DefaultClient,
		cleanup: func() {
			cancel()
			done := make(chan error, 1)
			go func() { done <- cmd.Wait() }()
			select {
			case <-done:
			case <-time.After(5 * time.Second):
				if cmd.Process != nil {
					_ = cmd.Process.Kill()
				}
				<-done
			}
		},
	}
}

func applyContentMigrations(t *testing.T, root string, dsn string) {
	t.Helper()

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open postgres for migration: %v", err)
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("ping postgres for migration: %v", err)
	}

	migrationPath := filepath.Join(root, "services", "zhicore-content", "migrations", "20260705093000_create_content_publish_core.up.sql")
	body, err := os.ReadFile(migrationPath)
	if err != nil {
		t.Fatalf("read content migration: %v", err)
	}
	if _, err := db.ExecContext(ctx, string(body)); err != nil {
		t.Fatalf("apply content migration %s: %v", migrationPath, err)
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get cwd: %v", err)
	}
	for dir := wd; ; dir = filepath.Dir(dir) {
		if fileExists(filepath.Join(dir, "go.work")) && fileExists(filepath.Join(dir, "services", "zhicore-content", "go.mod")) {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("repo root not found from %s", wd)
		}
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func reserveLocalAddr(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve local port: %v", err)
	}
	addr := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatalf("close reserved listener: %v", err)
	}
	return addr
}

func waitForReady(t *testing.T, client *http.Client, baseURL string, output fmt.Stringer) {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	var lastErr error
	for time.Now().Before(deadline) {
		resp, err := client.Get(baseURL + "/health/ready")
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
			lastErr = fmt.Errorf("ready status %d", resp.StatusCode)
		} else {
			lastErr = err
		}
		time.Sleep(200 * time.Millisecond)
	}
	if lastErr == nil {
		lastErr = errors.New("ready check timed out")
	}
	t.Fatalf("content server not ready: %v\nprocess output:\n%s", lastErr, output.String())
}

func newFakeUserServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != usercontract.BatchSimplePath {
			http.NotFound(w, r)
			return
		}
		var request usercontract.IDsRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		items := make([]usercontract.SimpleUser, 0, len(request.UserIDs))
		for _, userID := range request.UserIDs {
			items = append(items, usercontract.SimpleUser{
				UserID:         userID,
				PublicID:       fmt.Sprintf("user_%d", userID),
				Nickname:       fmt.Sprintf("User %d", userID),
				ProfileVersion: 1,
				Status:         "ACTIVE",
			})
		}
		writeProviderEnvelope(t, w, usercontract.SimpleBatchResponse{Items: items})
	}))
}

func newFakeFileServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != filecontract.ValidateRefsPath {
			http.NotFound(w, r)
			return
		}
		writeProviderEnvelope(t, w, filecontract.ValidateRefsResponse{})
	}))
}

func writeProviderEnvelope(t *testing.T, w http.ResponseWriter, data any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{
		"code":    200,
		"message": "操作成功",
		"data":    data,
	}); err != nil {
		t.Fatalf("write provider envelope: %v", err)
	}
}
