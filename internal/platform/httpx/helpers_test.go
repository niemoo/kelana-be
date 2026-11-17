package httpx

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"kelana/internal/platform/logx"
)

// newRequest membuat request test yang membawa logger "bisu", supaya output
// go test tidak dipenuhi log dari kode yang sedang dites.
func newRequest(t *testing.T, method, target string) *http.Request {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), method, target, nil)
	return req.WithContext(logx.WithLogger(req.Context(), slog.New(slog.DiscardHandler)))
}
