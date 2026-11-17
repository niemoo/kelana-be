package httpx

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWriteJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteJSON(rec, newRequest(t, http.MethodGet, "/"), http.StatusCreated, map[string]int{"seats_left": 9})

	if rec.Code != http.StatusCreated {
		t.Errorf("status = %d, want 201", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
	if body := rec.Body.String(); body != `{"seats_left":9}` {
		t.Errorf("body = %s", body)
	}
}

func TestWriteJSONEncodeErrorBecomes500(t *testing.T) {
	rec := httptest.NewRecorder()
	// Channel tidak bisa diubah menjadi JSON, jadi encoding pasti gagal.
	WriteJSON(rec, newRequest(t, http.MethodGet, "/"), http.StatusOK, make(chan int))

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
}

func TestWriteProblemFillsDefaults(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteProblem(rec, newRequest(t, http.MethodGet, "/"), Problem{
		Status: http.StatusConflict,
		Code:   "SOLD_OUT",
		Title:  "Kursi tidak cukup",
	})

	if rec.Code != http.StatusConflict {
		t.Errorf("status = %d, want 409", rec.Code)
	}
	if body := rec.Body.String(); !strings.Contains(body, `"type":"about:blank"`) || !strings.Contains(body, `"code":"SOLD_OUT"`) {
		t.Errorf("body = %s", body)
	}
}
