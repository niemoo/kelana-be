package database

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"
)

func TestAddr(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		want    string
		wantErr string
	}{
		{name: "lengkap", url: "postgres://kelana:kelana@localhost:5432/kelana?sslmode=disable", want: "localhost:5432"},
		{name: "tanpa port memakai 5432", url: "postgresql://db.internal/kelana", want: "db.internal:5432"},
		{name: "IPv6", url: "postgres://u:p@[::1]:6543/kelana", want: "[::1]:6543"},
		{name: "scheme lain", url: "mysql://root@localhost:3306/kelana", wantErr: `scheme "mysql"`},
		{name: "tanpa host", url: "postgres:///kelana", wantErr: "missing host"},
		{name: "bukan URL", url: "postgres://u:p@local host/kelana", wantErr: "invalid database URL"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Addr(tt.url)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("Addr(%q) error = %v, want error containing %q", tt.url, err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Addr(%q) unexpected error: %v", tt.url, err)
			}
			if got != tt.want {
				t.Errorf("Addr(%q) = %q, want %q", tt.url, got, tt.want)
			}
		})
	}
}

func TestAddrErrorDoesNotLeakPassword(t *testing.T) {
	_, err := Addr("postgres://kelana:rahasia@local host:5432/kelana")
	if err == nil {
		t.Fatal("Addr() error = nil, want parse error")
	}
	if strings.Contains(err.Error(), "rahasia") {
		t.Fatalf("error leaks the password: %v", err)
	}
}

func TestPingTCP(t *testing.T) {
	var lc net.ListenConfig
	ln, err := lc.Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	check, err := PingTCP("postgres://kelana:kelana@" + ln.Addr().String() + "/kelana")
	if err != nil {
		t.Fatalf("PingTCP() error: %v", err)
	}

	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	if err := check(ctx); err != nil {
		t.Errorf("check() with server listening = %v, want nil", err)
	}

	if err := ln.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	if err := check(ctx); err == nil {
		t.Error("check() with server stopped = nil, want error")
	}
}

func TestPingTCPRejectsInvalidURL(t *testing.T) {
	if _, err := PingTCP("mysql://localhost/kelana"); err == nil {
		t.Fatal("PingTCP() error = nil, want error for non-postgres URL")
	}
}
