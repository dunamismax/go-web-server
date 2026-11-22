package main

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dunamismax/go-web-server/internal/store/storetest"
)

func TestRunVersionAndHelp(t *testing.T) {
	t.Parallel()

	for _, arg := range []string{"version", "--version"} {
		var out bytes.Buffer
		if err := run(t.Context(), []string{arg}, &out); err != nil {
			t.Fatalf("run(%s): %v", arg, err)
		}
		if strings.TrimSpace(out.String()) == "" {
			t.Errorf("run(%s) printed nothing", arg)
		}
	}

	var out bytes.Buffer
	if err := run(t.Context(), []string{"help"}, &out); err != nil {
		t.Fatalf("run(help): %v", err)
	}
	if !strings.Contains(out.String(), "migrate up") {
		t.Errorf("help output does not list the commands: %q", out.String())
	}
}

func TestRunRejectsUnknownCommands(t *testing.T) {
	t.Parallel()

	err := run(t.Context(), []string{"frobnicate"}, io.Discard)
	if err == nil || !strings.Contains(err.Error(), `unknown command "frobnicate"`) {
		t.Errorf("run(frobnicate) error = %v; want an unknown-command error", err)
	}
}

func TestRunRequiresDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")

	err := run(t.Context(), nil, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "DATABASE_URL") {
		t.Errorf("run() without DATABASE_URL: error = %v; want one naming DATABASE_URL", err)
	}
}

// TestServeLifecycle boots the real entry point on a free port against an
// empty schema, checks that it migrated and serves, and shuts it down the
// way a SIGTERM would.
func TestServeLifecycle(t *testing.T) {
	t.Setenv("DATABASE_URL", storetest.NewURL(t))
	t.Setenv("HTTP_ADDR", "127.0.0.1:0")
	t.Setenv("APP_ENV", "production") // exercises JSON logs and embedded assets
	t.Setenv("SHUTDOWN_TIMEOUT", "10s")

	ctx, shutdown := context.WithCancel(t.Context())
	defer shutdown()

	logs := &syncBuffer{}
	done := make(chan error, 1)
	go func() { done <- run(ctx, []string{"serve"}, logs) }()

	addr := waitForListenAddr(t, logs, done)

	for _, path := range []string{"/healthz", "/readyz", "/", "/login"} {
		if status := getStatus(t, "http://"+addr+path); status != http.StatusOK {
			t.Errorf("GET %s = %d; want 200", path, status)
		}
	}

	// The healthcheck subcommand finds the server through HTTP_ADDR.
	t.Setenv("HTTP_ADDR", addr)
	if err := run(t.Context(), []string{"healthcheck"}, io.Discard); err != nil {
		t.Errorf("healthcheck against a running server: %v", err)
	}

	// The migrate subcommand sees the schema the server already migrated.
	var status bytes.Buffer
	if err := run(t.Context(), []string{"migrate", "status"}, &status); err != nil {
		t.Fatalf("migrate status: %v", err)
	}
	if !strings.Contains(status.String(), "applied") || strings.Contains(status.String(), "pending") {
		t.Errorf("migrate status after startup = %q; want everything applied", status.String())
	}

	shutdown()
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("serve returned %v after shutdown; want nil", err)
		}
	case <-time.After(15 * time.Second):
		t.Fatal("serve did not return within 15s of shutdown")
	}

	output := logs.String()
	for _, want := range []string{`"msg":"applied migration"`, `"msg":"shutting down"`, `"msg":"stopped"`} {
		if !strings.Contains(output, want) {
			t.Errorf("logs do not contain %s\nlogs:\n%s", want, output)
		}
	}

	if err := run(t.Context(), []string{"healthcheck"}, io.Discard); err == nil {
		t.Error("healthcheck succeeded against a stopped server")
	}
}

var listenAddr = regexp.MustCompile(`"msg":"listening","addr":"([^"]+)"`)

func waitForListenAddr(t *testing.T, logs *syncBuffer, done <-chan error) string {
	t.Helper()

	deadline := time.After(20 * time.Second)
	for {
		if m := listenAddr.FindStringSubmatch(logs.String()); m != nil {
			return m[1]
		}
		select {
		case err := <-done:
			t.Fatalf("serve exited before listening: %v\nlogs:\n%s", err, logs.String())
		case <-deadline:
			t.Fatalf("serve did not start listening in time\nlogs:\n%s", logs.String())
		case <-time.After(20 * time.Millisecond):
		}
	}
}

// probeClient closes its connection after every request. net/http keeps a
// just-opened, unused connection alive for up to five seconds during
// Shutdown, so a pooled client would make the shutdown assertion slow and
// timing-dependent.
var probeClient = &http.Client{Transport: &http.Transport{DisableKeepAlives: true}}

func getStatus(t *testing.T, target string) int {
	t.Helper()

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, target, http.NoBody)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	resp, err := probeClient.Do(req)
	if err != nil {
		t.Fatalf("GET %s: %v", target, err)
	}
	defer resp.Body.Close()

	return resp.StatusCode
}

// syncBuffer is a bytes.Buffer that is safe to write from the server
// goroutine while the test reads it.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}
