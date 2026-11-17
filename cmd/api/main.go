// Command api menjalankan HTTP API Kelana.
//
// Konfigurasi dibaca dari environment variable berprefix KELANA_ (lihat
// .env.example). Server berhenti dengan rapi saat menerima Ctrl+C (SIGINT)
// atau SIGTERM: request yang sedang berjalan diberi waktu selesai sampai
// KELANA_SHUTDOWN_TIMEOUT.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"kelana/internal/platform/config"
	"kelana/internal/platform/database"
	"kelana/internal/platform/logx"
)

func main() {
	if err := run(context.Background(), os.LookupEnv, os.Stdout); err != nil {
		// Logger belum tentu siap (mis. config gagal dibaca), jadi tulis ke stderr.
		fmt.Fprintln(os.Stderr, "kelana-api:", err)
		os.Exit(1)
	}
}

// run berisi seluruh isi program. Dipisah dari main supaya bisa dites: test
// memberi ctx yang bisa dibatalkan dan env palsu, tanpa os.Exit.
func run(ctx context.Context, lookupEnv config.LookupFunc, stdout io.Writer) error {
	cfg, err := config.Load(lookupEnv)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	logger := logx.New(stdout, cfg.LogLevel, cfg.LogFormat)
	slog.SetDefault(logger)
	logger.Info("config loaded", "config", cfg)

	dbCheck, err := database.PingTCP(cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("database check: %w", err)
	}

	// ctx batal saat Ctrl+C (SIGINT) atau SIGTERM (mis. dari docker stop).
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	srv := &http.Server{
		Handler:           routes(logger, dbCheck),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelWarn),
	}

	// Listen dulu, baru Serve. Kalau port sudah dipakai, error-nya muncul di
	// sini, dan "server listening" hanya tercetak bila port benar-benar terbuka.
	var lc net.ListenConfig
	ln, err := lc.Listen(ctx, "tcp", cfg.HTTPAddr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", cfg.HTTPAddr, err)
	}
	logger.Info("server listening", "addr", ln.Addr().String())

	serveErr := make(chan error, 1)
	go func() { serveErr <- srv.Serve(ln) }()

	select {
	case err := <-serveErr:
		// Serve berhenti tanpa diminta: itu error sungguhan.
		return fmt.Errorf("serve: %w", err)
	case <-ctx.Done():
	}

	// Kembalikan perilaku sinyal ke default: Ctrl+C kedua langsung mematikan proses.
	stop()
	logger.Info("shutting down", "timeout", cfg.ShutdownTimeout.String())

	// ctx sudah batal, jadi Shutdown butuh context baru dengan batas waktu.
	// WithoutCancel mewarisi isi ctx tanpa ikut batal.
	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cfg.ShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}

	// Setelah Shutdown, Serve selalu mengembalikan http.ErrServerClosed.
	// Menunggunya memastikan goroutine di atas sudah selesai.
	if err := <-serveErr; !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve: %w", err)
	}
	logger.Info("server stopped")
	return nil
}
