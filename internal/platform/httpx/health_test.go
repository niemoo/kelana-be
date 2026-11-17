package httpx

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestLiveness(t *testing.T) {
	rec := httptest.NewRecorder()
	Liveness().ServeHTTP(rec, newRequest(t, http.MethodGet, "/healthz"))

	if rec.Code != http.StatusOK || rec.Body.String() != `{"status":"ok"}` {
		t.Errorf("GET /healthz = %d %s, want 200 {\"status\":\"ok\"}", rec.Code, rec.Body.String())
	}
}

func TestReadiness(t *testing.T) {
	healthy := func(context.Context) error { return nil }
	broken := func(context.Context) error { return errors.New("connection refused") }
	slow := func(ctx context.Context) error { // hanya selesai saat batas waktu habis
		<-ctx.Done()
		return ctx.Err()
	}

	tests := []struct {
		name       string
		checks     map[string]Check
		wantStatus int
		wantBody   readinessBody
	}{
		{
			name:       "semua sehat",
			checks:     map[string]Check{"database": healthy, "redis": healthy},
			wantStatus: http.StatusOK,
			wantBody:   readinessBody{Status: "ok", Checks: map[string]string{"database": "ok", "redis": "ok"}},
		},
		{
			name:       "satu gagal",
			checks:     map[string]Check{"database": broken, "redis": healthy},
			wantStatus: http.StatusServiceUnavailable,
			wantBody:   readinessBody{Status: "unavailable", Checks: map[string]string{"database": "unavailable", "redis": "ok"}},
		},
		{
			name:       "melewati batas waktu",
			checks:     map[string]Check{"database": slow},
			wantStatus: http.StatusServiceUnavailable,
			wantBody:   readinessBody{Status: "unavailable", Checks: map[string]string{"database": "unavailable"}},
		},
		{
			name:       "tanpa check",
			checks:     nil,
			wantStatus: http.StatusOK,
			wantBody:   readinessBody{Status: "ok", Checks: map[string]string{}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			Readiness(50*time.Millisecond, tt.checks).ServeHTTP(rec, newRequest(t, http.MethodGet, "/readyz"))

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			var got readinessBody
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatalf("body is not JSON: %v", err)
			}
			if !reflect.DeepEqual(got, tt.wantBody) {
				t.Errorf("body = %+v, want %+v", got, tt.wantBody)
			}
		})
	}
}

func TestReadinessHidesErrorDetails(t *testing.T) {
	leaky := func(context.Context) error {
		return errors.New("password authentication failed for user kelana")
	}
	rec := httptest.NewRecorder()
	Readiness(time.Second, map[string]Check{"database": leaky}).ServeHTTP(rec, newRequest(t, http.MethodGet, "/readyz"))

	if strings.Contains(rec.Body.String(), "password") {
		t.Errorf("response leaks the check error: %s", rec.Body.String())
	}
}
