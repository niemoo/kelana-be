package httpx

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"regexp"
	"slices"
	"strings"
	"testing"
)

func TestChainRunsMiddlewareInOrder(t *testing.T) {
	var order []string
	mark := func(name string) Middleware {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				order = append(order, name+" masuk")
				next.ServeHTTP(w, r)
				order = append(order, name+" keluar")
			})
		}
	}
	h := Chain(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		order = append(order, "handler")
	}), mark("a"), mark("b"))

	h.ServeHTTP(httptest.NewRecorder(), newRequest(t, http.MethodGet, "/"))

	want := []string{"a masuk", "b masuk", "handler", "b keluar", "a keluar"}
	if !slices.Equal(order, want) {
		t.Errorf("order = %q, want %q", order, want)
	}
}

// rand.Text menghasilkan 26 karakter dari alfabet base32: A–Z dan 2–7.
var generatedID = regexp.MustCompile(`^[A-Z2-7]{26}$`)

func TestRequestID(t *testing.T) {
	tests := []struct {
		name     string
		incoming string
		wantKept bool
	}{
		{name: "dibuat bila klien tidak mengirim", incoming: ""},
		{name: "dipakai ulang bila formatnya aman", incoming: "abc-123_X.y", wantKept: true},
		{name: "diganti bila berisi spasi", incoming: "abc 123"},
		{name: "diganti bila berisi baris baru", incoming: "abc\nlevel=ERROR"},
		{name: "diganti bila terlalu panjang", incoming: strings.Repeat("a", 129)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var inContext string
			h := RequestID(slog.New(slog.DiscardHandler))(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				inContext = RequestIDFrom(r.Context())
			}))
			req := newRequest(t, http.MethodGet, "/")
			if tt.incoming != "" {
				req.Header.Set(HeaderRequestID, tt.incoming)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			got := rec.Header().Get(HeaderRequestID)
			if got != inContext {
				t.Errorf("header %q differs from context %q", got, inContext)
			}
			if tt.wantKept {
				if got != tt.incoming {
					t.Errorf("request ID = %q, want the client's %q", got, tt.incoming)
				}
				return
			}
			if !generatedID.MatchString(got) {
				t.Errorf("request ID = %q, want a newly generated ID", got)
			}
		})
	}
}

func TestAccessLog(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	h := Chain(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
		_, _ = w.Write([]byte("short and stout"))
	}), RequestID(logger), AccessLog())

	h.ServeHTTP(httptest.NewRecorder(), newRequest(t, http.MethodPost, "/v1/teapot"))

	var line map[string]any
	if err := json.Unmarshal(buf.Bytes(), &line); err != nil {
		t.Fatalf("log is not one JSON line: %v\n%s", err, buf.String())
	}
	want := map[string]any{
		"msg":    "http request",
		"method": "POST",
		"path":   "/v1/teapot",
		"status": float64(http.StatusTeapot),
		"bytes":  float64(len("short and stout")),
	}
	for k, v := range want {
		if line[k] != v {
			t.Errorf("log field %s = %v, want %v", k, line[k], v)
		}
	}
	for _, k := range []string{"request_id", "duration_ms"} {
		if _, ok := line[k]; !ok {
			t.Errorf("log line has no %s field: %v", k, line)
		}
	}
}

func TestAccessLogDefaultsTo200(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	h := Chain(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}), RequestID(logger), AccessLog())

	h.ServeHTTP(httptest.NewRecorder(), newRequest(t, http.MethodGet, "/"))

	if !strings.Contains(buf.String(), `"status":200`) {
		t.Errorf("handler that writes nothing should log status 200, got %s", buf.String())
	}
}

func TestRecoverer(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	h := Chain(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	}), RequestID(logger), Recoverer())

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, newRequest(t, http.MethodGet, "/"))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/problem+json" {
		t.Errorf("Content-Type = %q, want application/problem+json", ct)
	}
	var p Problem
	if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
		t.Fatalf("body is not JSON: %v", err)
	}
	if p.Code != "INTERNAL_ERROR" || p.Status != http.StatusInternalServerError {
		t.Errorf("problem = %+v, want code INTERNAL_ERROR and status 500", p)
	}
	if p.RequestID == "" || p.RequestID != rec.Header().Get(HeaderRequestID) {
		t.Errorf("problem request_id = %q, want the X-Request-Id header", p.RequestID)
	}
	if out := buf.String(); !strings.Contains(out, "panic recovered") || !strings.Contains(out, "boom") {
		t.Errorf("panic was not logged: %s", out)
	}
}

func TestRecovererLetsErrAbortHandlerThrough(t *testing.T) {
	h := Recoverer()(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic(http.ErrAbortHandler)
	}))
	defer func() {
		err, _ := recover().(error)
		if !errors.Is(err, http.ErrAbortHandler) {
			t.Errorf("recovered %v, want http.ErrAbortHandler to be re-panicked", err)
		}
	}()
	h.ServeHTTP(httptest.NewRecorder(), newRequest(t, http.MethodGet, "/"))
	t.Error("ServeHTTP returned normally, want a panic")
}

func TestStatusRecorderSupportsFlush(t *testing.T) {
	inner := httptest.NewRecorder()
	rec := &statusRecorder{ResponseWriter: inner}

	// ResponseController memakai Unwrap untuk menemukan Flush milik inner.
	if err := http.NewResponseController(rec).Flush(); err != nil {
		t.Fatalf("Flush() through statusRecorder: %v", err)
	}
	if !inner.Flushed {
		t.Error("inner ResponseWriter was not flushed")
	}
}
