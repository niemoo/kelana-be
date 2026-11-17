// Package logx menyiapkan logger slog untuk Kelana dan menitipkannya di
// context, supaya setiap baris log dalam satu request membawa request_id.
package logx

import (
	"context"
	"io"
	"log/slog"
)

// New membuat logger yang menulis ke w dan membuang log di bawah level.
// Format "text" menghasilkan key=value yang enak dibaca saat development;
// nilai lain menghasilkan JSON, yang mudah diolah tool pengumpul log.
func New(w io.Writer, level slog.Leveler, format string) *slog.Logger {
	opts := &slog.HandlerOptions{Level: level}
	if format == "text" {
		return slog.New(slog.NewTextHandler(w, opts))
	}
	return slog.New(slog.NewJSONHandler(w, opts))
}

// ctxKey bertipe sendiri (bukan string) supaya tidak mungkin bentrok dengan
// key milik package lain.
type ctxKey struct{}

// WithLogger mengembalikan salinan ctx yang membawa l.
func WithLogger(ctx context.Context, l *slog.Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, l)
}

// FromContext mengambil logger dari ctx. Bila tidak ada, slog.Default()
// dipakai, jadi pemanggil tidak perlu memeriksa nil.
func FromContext(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(ctxKey{}).(*slog.Logger); ok {
		return l
	}
	return slog.Default()
}
