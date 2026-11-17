package httpx

import (
	"context"
	"log/slog"
	"maps"
	"net/http"
	"slices"
	"time"

	"kelana/internal/platform/logx"
)

// Check memeriksa satu dependency (database, Redis, ...); nil berarti sehat.
// Check wajib berhenti saat ctx batal, supaya batas waktunya berlaku.
type Check func(ctx context.Context) error

// Liveness menjawab 200 selama proses hidup dan bisa melayani HTTP. Endpoint
// ini sengaja tidak memeriksa dependency: database mati bukan alasan untuk
// me-restart API.
func Liveness() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		WriteJSON(w, r, http.StatusOK, map[string]string{"status": "ok"})
	})
}

type readinessBody struct {
	Status string            `json:"status"`
	Checks map[string]string `json:"checks"`
}

// Readiness menjalankan setiap check dengan batas waktu timeout. Semua lolos
// menghasilkan 200; satu saja gagal menghasilkan 503, tanda load balancer
// berhenti mengirim traffic. Detail error hanya masuk log, tidak ke klien.
func Readiness(timeout time.Duration, checks map[string]Check) http.Handler {
	checks = maps.Clone(checks) // salinan: map milik pemanggil boleh berubah
	names := slices.Sorted(maps.Keys(checks))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := readinessBody{Status: "ok", Checks: make(map[string]string, len(names))}
		for _, name := range names {
			if err := runCheck(r.Context(), timeout, checks[name]); err != nil {
				logx.FromContext(r.Context()).Warn("readiness check failed",
					slog.String("check", name),
					slog.Any("error", err),
				)
				body.Checks[name] = "unavailable"
				body.Status = "unavailable"
				continue
			}
			body.Checks[name] = "ok"
		}

		status := http.StatusOK
		if body.Status != "ok" {
			status = http.StatusServiceUnavailable
		}
		WriteJSON(w, r, status, body)
	})
}

func runCheck(ctx context.Context, timeout time.Duration, check Check) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return check(ctx)
}
