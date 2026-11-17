// Package config membaca dan memvalidasi konfigurasi Kelana dari environment
// variable berprefix KELANA_. Semua kesalahan dikumpulkan sekaligus, supaya
// program gagal cepat saat start dan kamu bisa memperbaiki semuanya dalam satu
// kali jalan.
package config

import (
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"time"

	"kelana/internal/platform/database"
)

// Config adalah konfigurasi yang dibutuhkan cmd/api di Fase 0.
type Config struct {
	HTTPAddr        string        // KELANA_HTTP_ADDR, default ":8080"
	DatabaseURL     string        // KELANA_DATABASE_URL, wajib; berisi password
	LogLevel        slog.Level    // KELANA_LOG_LEVEL: debug, info, warn, error; default info
	LogFormat       string        // KELANA_LOG_FORMAT: json atau text; default json
	ShutdownTimeout time.Duration // KELANA_SHUTDOWN_TIMEOUT, default 10s
}

// LookupFunc mencari sebuah environment variable. os.LookupEnv memenuhi tipe
// ini, dan test bisa memakai map biasa.
type LookupFunc func(key string) (string, bool)

// Load membaca konfigurasi lewat lookup, mengisi nilai default, lalu
// memvalidasinya. Variabel yang kosong dianggap tidak diisi.
func Load(lookup LookupFunc) (Config, error) {
	cfg := Config{
		HTTPAddr:        ":8080",
		LogLevel:        slog.LevelInfo,
		LogFormat:       "json",
		ShutdownTimeout: 10 * time.Second,
	}
	get := func(key string) string {
		v, _ := lookup(key)
		return v
	}
	var errs []error

	if v := get("KELANA_HTTP_ADDR"); v != "" {
		cfg.HTTPAddr = v
	}
	if _, _, err := net.SplitHostPort(cfg.HTTPAddr); err != nil {
		errs = append(errs, fmt.Errorf("KELANA_HTTP_ADDR %q: want host:port such as :8080", cfg.HTTPAddr))
	}

	cfg.DatabaseURL = get("KELANA_DATABASE_URL")
	if cfg.DatabaseURL == "" {
		errs = append(errs, errors.New("KELANA_DATABASE_URL is required, e.g. postgres://kelana:kelana@localhost:5432/kelana?sslmode=disable"))
	} else if _, err := database.Addr(cfg.DatabaseURL); err != nil {
		errs = append(errs, fmt.Errorf("KELANA_DATABASE_URL: %w", err))
	}

	if v := get("KELANA_LOG_LEVEL"); v != "" {
		if err := cfg.LogLevel.UnmarshalText([]byte(v)); err != nil {
			errs = append(errs, fmt.Errorf("KELANA_LOG_LEVEL %q: want debug, info, warn, or error", v))
		}
	}

	if v := get("KELANA_LOG_FORMAT"); v != "" {
		cfg.LogFormat = v
	}
	if cfg.LogFormat != "json" && cfg.LogFormat != "text" {
		errs = append(errs, fmt.Errorf("KELANA_LOG_FORMAT %q: want json or text", cfg.LogFormat))
	}

	if v := get("KELANA_SHUTDOWN_TIMEOUT"); v != "" {
		d, err := time.ParseDuration(v)
		switch {
		case err != nil:
			errs = append(errs, fmt.Errorf("KELANA_SHUTDOWN_TIMEOUT %q: want a duration such as 10s", v))
		case d <= 0:
			errs = append(errs, fmt.Errorf("KELANA_SHUTDOWN_TIMEOUT %q: must be greater than zero", v))
		default:
			cfg.ShutdownTimeout = d
		}
	}

	if err := errors.Join(errs...); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// LogValue membuat Config aman untuk di-log: password di KELANA_DATABASE_URL
// disamarkan. Method ini memenuhi interface slog.LogValuer, jadi cukup tulis
// logger.Info("config loaded", "config", cfg).
func (c Config) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("http_addr", c.HTTPAddr),
		slog.String("database_url", redactURL(c.DatabaseURL)),
		slog.String("log_level", c.LogLevel.String()),
		slog.String("log_format", c.LogFormat),
		slog.String("shutdown_timeout", c.ShutdownTimeout.String()),
	)
}

// redactURL mengganti password di URL dengan "xxxxx".
func redactURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return "[invalid URL]"
	}
	return u.Redacted()
}
