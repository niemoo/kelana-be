package config

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func lookupFrom(env map[string]string) LookupFunc {
	return func(key string) (string, bool) {
		v, ok := env[key]
		return v, ok
	}
}

func TestLoad(t *testing.T) {
	const dbURL = "postgres://kelana:kelana@localhost:5432/kelana?sslmode=disable"
	defaults := Config{
		HTTPAddr:        ":8080",
		DatabaseURL:     dbURL,
		LogLevel:        slog.LevelInfo,
		LogFormat:       "json",
		ShutdownTimeout: 10 * time.Second,
	}

	tests := []struct {
		name    string
		env     map[string]string
		want    Config
		wantErr []string // potongan teks yang wajib ada di pesan error
	}{
		{
			name: "default dipakai bila hanya database URL yang diisi",
			env:  map[string]string{"KELANA_DATABASE_URL": dbURL},
			want: defaults,
		},
		{
			name: "nilai kosong dianggap tidak diisi",
			env: map[string]string{
				"KELANA_DATABASE_URL": dbURL,
				"KELANA_HTTP_ADDR":    "",
				"KELANA_LOG_LEVEL":    "",
			},
			want: defaults,
		},
		{
			name: "semua nilai diisi",
			env: map[string]string{
				"KELANA_HTTP_ADDR":        "127.0.0.1:9090",
				"KELANA_DATABASE_URL":     dbURL,
				"KELANA_LOG_LEVEL":        "DEBUG",
				"KELANA_LOG_FORMAT":       "text",
				"KELANA_SHUTDOWN_TIMEOUT": "3s",
			},
			want: Config{
				HTTPAddr:        "127.0.0.1:9090",
				DatabaseURL:     dbURL,
				LogLevel:        slog.LevelDebug,
				LogFormat:       "text",
				ShutdownTimeout: 3 * time.Second,
			},
		},
		{
			name:    "database URL wajib diisi",
			env:     map[string]string{},
			wantErr: []string{"KELANA_DATABASE_URL", "required"},
		},
		{
			name:    "database selain PostgreSQL ditolak",
			env:     map[string]string{"KELANA_DATABASE_URL": "mysql://root@localhost:3306/kelana"},
			wantErr: []string{"KELANA_DATABASE_URL", "mysql"},
		},
		{
			name: "durasi tanpa satuan ditolak",
			env: map[string]string{
				"KELANA_DATABASE_URL":     dbURL,
				"KELANA_SHUTDOWN_TIMEOUT": "10",
			},
			wantErr: []string{"KELANA_SHUTDOWN_TIMEOUT"},
		},
		{
			name: "semua kesalahan dilaporkan sekaligus",
			env: map[string]string{
				"KELANA_HTTP_ADDR":        "8080",
				"KELANA_LOG_LEVEL":        "verbose",
				"KELANA_LOG_FORMAT":       "xml",
				"KELANA_SHUTDOWN_TIMEOUT": "-1s",
			},
			wantErr: []string{
				"KELANA_HTTP_ADDR",
				"KELANA_DATABASE_URL",
				"KELANA_LOG_LEVEL",
				"KELANA_LOG_FORMAT",
				"KELANA_SHUTDOWN_TIMEOUT",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Load(lookupFrom(tt.env))
			if len(tt.wantErr) > 0 {
				if err == nil {
					t.Fatalf("Load() error = nil, want error mentioning %q", tt.wantErr)
				}
				for _, s := range tt.wantErr {
					if !strings.Contains(err.Error(), s) {
						t.Errorf("Load() error %q does not mention %q", err, s)
					}
				}
				return
			}
			if err != nil {
				t.Fatalf("Load() unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("Load() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestLogValueHidesPassword(t *testing.T) {
	cfg := Config{
		HTTPAddr:        ":8080",
		DatabaseURL:     "postgres://kelana:rahasia@localhost:5432/kelana",
		LogLevel:        slog.LevelInfo,
		LogFormat:       "json",
		ShutdownTimeout: 10 * time.Second,
	}
	var buf bytes.Buffer
	slog.New(slog.NewJSONHandler(&buf, nil)).Info("config loaded", "config", cfg)

	out := buf.String()
	if strings.Contains(out, "rahasia") {
		t.Fatalf("log leaks the database password: %s", out)
	}
	for _, want := range []string{`"http_addr":":8080"`, "localhost:5432", `"shutdown_timeout":"10s"`} {
		if !strings.Contains(out, want) {
			t.Errorf("log %s does not contain %s", out, want)
		}
	}
}
