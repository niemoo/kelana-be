package main

import (
	"log/slog"
	"net/http"
	"time"

	"kelana/internal/platform/httpx"
)

// readinessTimeout membatasi waktu setiap check di /readyz.
const readinessTimeout = 2 * time.Second

// routes merakit semua endpoint dan middleware. Mulai Fase 1, handler modul
// (identity, catalog, ...) juga didaftarkan di sini.
func routes(logger *slog.Logger, dbCheck httpx.Check) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /healthz", httpx.Liveness())
	mux.Handle("GET /readyz", httpx.Readiness(readinessTimeout, map[string]httpx.Check{
		"database": dbCheck,
	}))

	// Dibaca dari luar ke dalam: RequestID berjalan paling dulu, lalu
	// AccessLog, lalu Recoverer tepat sebelum handler.
	return httpx.Chain(mux,
		httpx.RequestID(logger),
		httpx.AccessLog(),
		httpx.Recoverer(),
	)
}
