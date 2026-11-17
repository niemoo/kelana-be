package httpx

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"kelana/internal/platform/logx"
)

// WriteJSON menulis v sebagai JSON dengan status code status. JSON disusun
// dulu di memori, jadi bila encoding gagal klien menerima 500, bukan respons
// yang terpotong di tengah.
func WriteJSON(w http.ResponseWriter, r *http.Request, status int, v any) {
	body, err := json.Marshal(v)
	if err != nil {
		logx.FromContext(r.Context()).Error("encode JSON response", slog.Any("error", err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	write(w, r, status, body)
}

// Problem adalah body error berformat RFC 9457 (application/problem+json).
// Code adalah kode stabil tempat frontend bercabang; daftarnya di tab 05.
type Problem struct {
	Type      string `json:"type"`
	Title     string `json:"title"`
	Status    int    `json:"status"`
	Detail    string `json:"detail,omitempty"`
	Code      string `json:"code"`
	RequestID string `json:"request_id,omitempty"`
}

// WriteProblem menulis p sebagai application/problem+json. Type diisi
// "about:blank" bila kosong, dan RequestID diambil dari context.
func WriteProblem(w http.ResponseWriter, r *http.Request, p Problem) {
	if p.Type == "" {
		p.Type = "about:blank"
	}
	if p.RequestID == "" {
		p.RequestID = RequestIDFrom(r.Context())
	}
	body, err := json.Marshal(p)
	if err != nil {
		logx.FromContext(r.Context()).Error("encode problem response", slog.Any("error", err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/problem+json")
	write(w, r, p.Status, body)
}

func write(w http.ResponseWriter, r *http.Request, status int, body []byte) {
	w.WriteHeader(status)
	if _, err := w.Write(body); err != nil {
		// Biasanya klien sudah memutus koneksi; tidak ada lagi yang bisa dikirim.
		logx.FromContext(r.Context()).Debug("write response", slog.Any("error", err))
	}
}
