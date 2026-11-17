package main

import (
	"context"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

// fakeEnv membuat pengganti os.LookupEnv dari sebuah map.
func fakeEnv(env map[string]string) func(string) (string, bool) {
	return func(key string) (string, bool) {
		v, ok := env[key]
		return v, ok
	}
}

// listen membuka port acak di localhost.
func listen(t *testing.T) net.Listener {
	t.Helper()
	var lc net.ListenConfig
	ln, err := lc.Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	return ln
}

// freeAddr meminjam port kosong dari OS lalu melepasnya lagi.
func freeAddr(t *testing.T) string {
	t.Helper()
	ln := listen(t)
	addr := ln.Addr().String()
	if err := ln.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	return addr
}

// fakeDatabase membuka port TCP yang menerima lalu langsung menutup koneksi.
// Itu cukup untuk lolos check database di Fase 0, yang hanya mencoba konek.
func fakeDatabase(t *testing.T) net.Listener {
	t.Helper()
	ln := listen(t)
	t.Cleanup(func() { _ = ln.Close() })
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return // listener ditutup
			}
			_ = conn.Close()
		}
	}()
	return ln
}

// client tanpa keep-alive, supaya setiap request membuka koneksi baru.
var client = &http.Client{
	Timeout:   2 * time.Second,
	Transport: &http.Transport{DisableKeepAlives: true},
}

// tryGet mengirim GET dan mengembalikan status serta header-nya. Body dibaca
// habis lalu ditutup di sini, jadi pemanggil tidak perlu mengurusnya.
func tryGet(t *testing.T, url string) (int, http.Header, error) {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	if _, err := io.Copy(io.Discard, resp.Body); err != nil {
		return 0, nil, err
	}
	return resp.StatusCode, resp.Header, nil
}

func mustGet(t *testing.T, url string) (int, http.Header) {
	t.Helper()
	status, header, err := tryGet(t, url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	return status, header
}

// waitForReady menunggu sampai server menjawab, karena run berjalan di goroutine.
func waitForReady(t *testing.T, url string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if status, _, err := tryGet(t, url); err == nil && status == http.StatusOK {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("server at %s not ready after 5s", url)
}

func TestRunServesHealthAndShutsDownGracefully(t *testing.T) {
	db := fakeDatabase(t)
	addr := freeAddr(t)
	env := fakeEnv(map[string]string{
		"KELANA_HTTP_ADDR":        addr,
		"KELANA_DATABASE_URL":     "postgres://kelana:kelana@" + db.Addr().String() + "/kelana?sslmode=disable",
		"KELANA_LOG_LEVEL":        "error",
		"KELANA_SHUTDOWN_TIMEOUT": "5s",
	})

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- run(ctx, env, io.Discard) }()

	base := "http://" + addr
	waitForReady(t, base+"/healthz")

	status, header := mustGet(t, base+"/readyz")
	if status != http.StatusOK {
		t.Errorf("GET /readyz with database up = %d, want 200", status)
	}
	if header.Get("X-Request-Id") == "" {
		t.Error("response has no X-Request-Id header")
	}

	// Database "mati": readiness ikut gagal, liveness tetap sehat.
	if err := db.Close(); err != nil {
		t.Fatalf("close fake database: %v", err)
	}
	if got, _ := mustGet(t, base+"/readyz"); got != http.StatusServiceUnavailable {
		t.Errorf("GET /readyz with database down = %d, want 503", got)
	}
	if got, _ := mustGet(t, base+"/healthz"); got != http.StatusOK {
		t.Errorf("GET /healthz with database down = %d, want 200", got)
	}

	cancel() // sama seperti menekan Ctrl+C
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("run() error = %v, want nil after graceful shutdown", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("run() did not return within 5s after cancel")
	}

	if _, _, err := tryGet(t, base+"/healthz"); err == nil {
		t.Error("server still answers after shutdown")
	}
}

func TestRunReturnsConfigError(t *testing.T) {
	err := run(t.Context(), fakeEnv(nil), io.Discard)
	if err == nil || !strings.Contains(err.Error(), "KELANA_DATABASE_URL") {
		t.Fatalf("run() error = %v, want error mentioning KELANA_DATABASE_URL", err)
	}
}

func TestRunReturnsErrorWhenPortIsTaken(t *testing.T) {
	taken := listen(t)
	defer taken.Close()

	env := fakeEnv(map[string]string{
		"KELANA_HTTP_ADDR":    taken.Addr().String(),
		"KELANA_DATABASE_URL": "postgres://kelana:kelana@127.0.0.1:5432/kelana",
		"KELANA_LOG_LEVEL":    "error",
	})
	err := run(t.Context(), env, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "listen on") {
		t.Fatalf("run() error = %v, want listen error", err)
	}
}
