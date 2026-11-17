// Package database nantinya berisi pool koneksi pgx dan helper transaksi
// (Fase 1). Di Fase 0 isinya baru satu hal: memeriksa bahwa server PostgreSQL
// bisa dihubungi, untuk endpoint /readyz.
package database

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
)

// DefaultPort dipakai bila URL tidak menyebutkan port.
const DefaultPort = "5432"

// Addr mengambil "host:port" dari URL PostgreSQL seperti
// postgres://kelana:kelana@localhost:5432/kelana. Pesan error sengaja tidak
// memuat URL-nya, karena URL itu berisi password.
func Addr(databaseURL string) (string, error) {
	u, err := url.Parse(databaseURL)
	if err != nil {
		// *url.Error menyimpan URL lengkap di pesannya; ambil penyebabnya saja.
		var urlErr *url.Error
		if errors.As(err, &urlErr) {
			err = urlErr.Err
		}
		return "", fmt.Errorf("invalid database URL: %w", err)
	}
	if u.Scheme != "postgres" && u.Scheme != "postgresql" {
		return "", fmt.Errorf("invalid database URL: scheme %q, want postgres", u.Scheme)
	}
	host := u.Hostname()
	if host == "" {
		return "", errors.New("invalid database URL: missing host")
	}
	port := u.Port()
	if port == "" {
		port = DefaultPort
	}
	return net.JoinHostPort(host, port), nil
}

// PingTCP mengembalikan fungsi check yang membuka koneksi TCP ke server
// database lalu langsung menutupnya. Ini hanya membuktikan port-nya terbuka.
// Di Fase 1, check ini diganti pool.Ping dari pgx yang benar-benar login.
func PingTCP(databaseURL string) (func(context.Context) error, error) {
	addr, err := Addr(databaseURL)
	if err != nil {
		return nil, err
	}
	return func(ctx context.Context) error {
		var d net.Dialer
		conn, err := d.DialContext(ctx, "tcp", addr)
		if err != nil {
			return fmt.Errorf("dial database %s: %w", addr, err)
		}
		return conn.Close()
	}, nil
}
