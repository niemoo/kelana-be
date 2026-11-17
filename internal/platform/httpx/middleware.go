// Package httpx berisi middleware HTTP dan helper respons yang dipakai semua
// modul. Package ini bagian dari platform: tidak boleh meng-import modul
// bisnis (dicek depguard di .golangci.yml).
package httpx

import (
	"context"
	"crypto/rand"
	"errors"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"kelana/internal/platform/logx"
)

// Middleware membungkus sebuah handler dengan perilaku tambahan.
type Middleware func(http.Handler) http.Handler

// Chain memasang middleware ke h. Chain(h, a, b) sama dengan a(b(h)):
// middleware pertama berjalan paling dulu dan selesai paling akhir.
func Chain(h http.Handler, mws ...Middleware) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}

// HeaderRequestID adalah header untuk menghubungkan log satu request di
// beberapa sistem.
const HeaderRequestID = "X-Request-Id"

type requestIDKey struct{}

// RequestIDFrom mengembalikan request ID yang dipasang middleware RequestID,
// atau string kosong bila tidak ada.
func RequestIDFrom(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey{}).(string)
	return id
}

// RequestID memakai X-Request-Id dari klien bila formatnya aman, atau membuat
// yang baru. ID itu dikirim balik di header respons, disimpan di context, dan
// ditempel ke logger turunan base supaya muncul di setiap baris log request.
func RequestID(base *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := r.Header.Get(HeaderRequestID)
			if !validRequestID(id) {
				id = rand.Text() // 26 karakter acak dari crypto/rand
			}
			w.Header().Set(HeaderRequestID, id)

			ctx := context.WithValue(r.Context(), requestIDKey{}, id)
			ctx = logx.WithLogger(ctx, base.With("request_id", id))
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// validRequestID menerima 1–128 karakter huruf, angka, '-', '_', atau '.'.
// ID dari luar dibatasi begini supaya tidak bisa menyisipkan teks aneh ke log.
func validRequestID(id string) bool {
	if id == "" || len(id) > 128 {
		return false
	}
	for _, c := range id {
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
		case c == '-' || c == '_' || c == '.':
		default:
			return false
		}
	}
	return true
}

// AccessLog menulis satu baris log per request setelah handler selesai,
// berisi method, path, status, jumlah byte, dan durasi.
func AccessLog() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w}
			next.ServeHTTP(rec, r)

			level := slog.LevelInfo
			if rec.Status() >= http.StatusInternalServerError {
				level = slog.LevelError
			}
			logx.FromContext(r.Context()).LogAttrs(r.Context(), level, "http request",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", rec.Status()),
				slog.Int("bytes", rec.bytes),
				slog.Int64("duration_ms", time.Since(start).Milliseconds()),
			)
		})
	}
}

// Recoverer mengubah panic di handler menjadi respons 500 dan satu log error
// lengkap dengan stack trace, supaya satu bug tidak mematikan seluruh server.
func Recoverer() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				rec := recover()
				if rec == nil {
					return
				}
				// http.ErrAbortHandler adalah cara resmi membatalkan respons; biarkan lewat.
				if err, ok := rec.(error); ok && errors.Is(err, http.ErrAbortHandler) {
					panic(rec)
				}
				logx.FromContext(r.Context()).Error("panic recovered",
					slog.Any("panic", rec),
					slog.String("stack", string(debug.Stack())),
				)
				WriteProblem(w, r, Problem{
					Status: http.StatusInternalServerError,
					Code:   "INTERNAL_ERROR",
					Title:  "Internal Server Error",
				})
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// statusRecorder membungkus http.ResponseWriter untuk mencatat status code
// dan jumlah byte yang ditulis handler.
type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (r *statusRecorder) WriteHeader(code int) {
	if r.status == 0 && code >= http.StatusOK { // abaikan status 1xx sementara
		r.status = code
	}
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK // Write tanpa WriteHeader berarti 200
	}
	n, err := r.ResponseWriter.Write(b)
	r.bytes += n
	return n, err
}

// Status mengembalikan status yang terkirim; handler yang tidak menulis apa
// pun tetap menghasilkan 200.
func (r *statusRecorder) Status() int {
	if r.status == 0 {
		return http.StatusOK
	}
	return r.status
}

// Unwrap membuat http.ResponseController tetap bisa menjangkau
// ResponseWriter asli (mis. untuk Flush) walaupun sedang dibungkus.
func (r *statusRecorder) Unwrap() http.ResponseWriter {
	return r.ResponseWriter
}
