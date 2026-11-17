# Panduan Setup Starter Project Kelana (Fase 0), Langkah demi Langkah

Panduan ini membangun ulang `kelana-example` dari folder kosong, **sedikit demi sedikit**. Setiap langkah diakhiri sesuatu yang bisa kamu jalankan, jadi kamu selalu tahu kodenya masih benar.

Aturan main:

- **Ketik sendiri, jangan copy-paste** dari `../kelana-example`. Pakai contoh itu hanya sebagai kunci jawaban setelah kamu mencoba.
- Selesaikan bagian **Cek** sebelum lanjut ke langkah berikutnya.
- Bagian **Konsep** adalah hal yang harus bisa kamu jelaskan tanpa membuka kode (Definition of Done di tab 07).
- Commit di akhir setiap langkah. Riwayat git-mu nanti jadi catatan belajar.

Peta langkah:

| # | Langkah | Hasil yang bisa dicek |
| --- | --- | --- |
| 0 | Prasyarat | semua tool terpasang |
| 1 | Repo + Go module | `go run` mencetak teks |
| 2 | Server HTTP pertama | `curl /healthz` → 200 |
| 3 | Makefile | `make run-api`, `make help` |
| 4 | Package `config` + test pertama | `make test` hijau |
| 5 | Package `logx` (slog) | log rapi di terminal |
| 6 | `run()` + graceful shutdown | Ctrl+C mematikan server dengan rapi |
| 7 | `httpx`: JSON + problem+json | respons JSON konsisten |
| 8 | Middleware + `routes.go` | header `X-Request-Id`, access log |
| 9 | PostgreSQL via Docker Compose | `make up` sehat |
| 10 | Package `database` + `/readyz` | `/readyz` 200 / 503 |
| 11 | Melengkapi test | `make test` hijau dengan `-race` |
| 12 | Linter + govulncheck | `make lint` hijau |
| 13 | CI GitHub Actions | CI hijau di GitHub |
| 14 | Playground latihan | `go test` di playground hijau |
| 15 | README + checklist Fase 0 | Fase 0 selesai |

---

## Langkah 0 · Prasyarat

Di laptopmu sudah ada Go 1.27.4, Docker Compose, make, dan git. Yang belum ada: **golangci-lint**.

```sh
brew install golangci-lint      # butuh v2.13 atau lebih baru untuk Go 1.27
golangci-lint version
```

Pasang juga ekstensi Go di VS Code (dari tim Go) supaya `gofmt`/`goimports` jalan otomatis saat simpan.

**Cek**

```sh
go version && docker compose version && make --version && golangci-lint version
```

---

## Langkah 1 · Repo dan Go module

```sh
cd /Volumes/T7Shield/dev/projects/go/kelanabe
git init -b main
go mod init kelana
```

Isi `go.mod` sekarang:

```text
module kelana

go 1.27
```

Buat `.gitignore`:

```gitignore
# Hasil build
/bin/

# Konfigurasi lokal, bisa berisi rahasia
.env

# Laporan coverage
*.out
coverage.html

# Editor dan OS
.idea/
.vscode/
.DS_Store
```

Buat struktur folder sesuai tab 07 (cukup yang dipakai Fase 0):

```sh
mkdir -p cmd/api internal/platform/{config,database,httpx,logx} deploy docs
```

Program pertama, `cmd/api/main.go`:

```go
// Command api menjalankan HTTP API Kelana.
package main

import "fmt"

func main() {
	fmt.Println("kelana-api: hello")
}
```

**Cek**

```sh
go run ./cmd/api          # kelana-api: hello
git add . && git commit -m "chore: init go module"
```

**Konsep**

- *Module* (`go.mod`, nama `kelana`) berisi banyak *package* (satu folder = satu package).
- `cmd/<nama>/` = satu binary (`package main`). `internal/` = kode yang **tidak bisa** di-import dari luar module ini; compiler yang menjaganya.
- Import path package dalam module = nama module + folder, mis. `kelana/internal/platform/config`.
- Bacaan: [Organizing a Go module](https://go.dev/doc/modules/layout).

---

## Langkah 2 · Server HTTP pertama

Ganti `cmd/api/main.go` dengan versi paling sederhana:

```go
package main

import (
	"log"
	"net/http"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
```

**Cek** (dua terminal)

```sh
go run ./cmd/api
curl -i localhost:8080/healthz     # 200 {"status":"ok"}
curl -i -X POST localhost:8080/healthz   # 405, karena pola "GET /healthz"
```

**Konsep**

- `http.Handler` adalah interface dengan satu method `ServeHTTP(w, r)`. `http.HandlerFunc` mengubah fungsi biasa menjadi `Handler`.
- `ServeMux` sejak Go 1.22 mengerti method dan path parameter (`"GET /bookings/{id}"`). Bacaan: [Routing Enhancements](https://go.dev/blog/routing-enhancements).
- Versi ini punya banyak kekurangan (tanpa timeout, tanpa shutdown rapi, config di-hardcode). Langkah 4–8 memperbaikinya satu per satu.

---

## Langkah 3 · Makefile

Semua perintah harian lewat `make`, supaya tidak ada langkah yang hanya tersimpan di kepala. Mulai dari yang kecil:

```makefile
.DEFAULT_GOAL := help
.PHONY: help run-api build test tidy clean

help: ## Tampilkan daftar target
	@awk 'BEGIN {FS = ":.*## "} /^[a-zA-Z_-]+:.*## / {printf "  %-10s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

run-api: build ## Build lalu jalankan API (Ctrl+C untuk berhenti)
	./bin/kelana-api

build: ## Build binary ke bin/kelana-api
	go build -o bin/kelana-api ./cmd/api

test: ## Unit test dengan race detector
	go test -race -count=1 ./...

tidy: ## Rapikan go.mod dan go.sum
	go mod tidy

clean: ## Hapus hasil build
	rm -rf bin
```

> Indentasi di bawah target **wajib TAB**, bukan spasi. Error `missing separator` hampir selalu karena ini.

**Cek**

```sh
make            # daftar target
make run-api    # server jalan dari bin/kelana-api
```

**Konsep**

- Kenapa `run-api` memakai binary hasil build dan bukan `go run`? `go run` keluar dengan status 1 setiap menerima Ctrl+C, jadi make melaporkan `Error 1` walau shutdown-nya bersih. Kamu baru akan merasakannya di Langkah 6.
- `-race` menyalakan race detector; `-count=1` mematikan cache hasil test.
- Target `up`, `down`, `logs`, `lint`, `vuln` ditambahkan nanti, saat kebutuhannya muncul.

---

## Langkah 4 · Package `config` dan test pertama

Tujuan: semua konfigurasi dari env berprefix `KELANA_`, divalidasi saat start, dan **semua** kesalahan dilaporkan sekaligus.

Buat `internal/platform/config/config.go`. Kerangkanya:

```go
// Package config membaca dan memvalidasi konfigurasi Kelana dari environment
// variable berprefix KELANA_.
package config

type Config struct {
	HTTPAddr        string        // KELANA_HTTP_ADDR, default ":8080"
	LogLevel        slog.Level    // KELANA_LOG_LEVEL, default info
	LogFormat       string        // KELANA_LOG_FORMAT: json | text, default json
	ShutdownTimeout time.Duration // KELANA_SHUTDOWN_TIMEOUT, default 10s
	// DatabaseURL ditambahkan di Langkah 10.
}

// LookupFunc mencari env var. os.LookupEnv cocok dengan tipe ini,
// dan test bisa memakai map biasa.
type LookupFunc func(key string) (string, bool)

func Load(lookup LookupFunc) (Config, error) {
	cfg := Config{ /* isi default */ }
	var errs []error
	// untuk setiap variabel: baca → bila kosong pakai default → validasi
	//   salah? errs = append(errs, fmt.Errorf("KELANA_X %q: want ...", v))
	return cfg, errors.Join(errs...) // nil bila errs kosong
}
```

Petunjuk validasi:

- alamat HTTP: `net.SplitHostPort`
- log level: `cfg.LogLevel.UnmarshalText([]byte(v))`
- durasi: `time.ParseDuration`, lalu tolak nilai `<= 0`

Lalu test table-driven di `config_test.go` (package yang sama):

```go
func lookupFrom(env map[string]string) LookupFunc {
	return func(k string) (string, bool) { v, ok := env[k]; return v, ok }
}

func TestLoad(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		want    Config
		wantErr []string // potongan teks yang harus muncul di error
	}{
		{name: "defaults", env: map[string]string{}, want: Config{HTTPAddr: ":8080", /* ... */}},
		{name: "bad level and duration", env: map[string]string{
			"KELANA_LOG_LEVEL": "loud", "KELANA_SHUTDOWN_TIMEOUT": "soon",
		}, wantErr: []string{"KELANA_LOG_LEVEL", "KELANA_SHUTDOWN_TIMEOUT"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) { /* panggil Load, bandingkan */ })
	}
}
```

Sambungkan ke `main.go`: `cfg, err := config.Load(os.LookupEnv)` lalu pakai `cfg.HTTPAddr`.

Buat `.env.example` dan bacakan dari Makefile:

```sh
# .env.example
KELANA_HTTP_ADDR=:8080
KELANA_LOG_LEVEL=debug
KELANA_LOG_FORMAT=text
KELANA_SHUTDOWN_TIMEOUT=10s
```

Tambahkan di **atas** Makefile:

```makefile
include .env.example
-include .env
export
```

**Cek**

```sh
make test
KELANA_LOG_LEVEL=loud KELANA_HTTP_ADDR=8080 make run-api   # dua error sekaligus, lalu exit 1
```

**Konsep**

- Error sebagai nilai, `errors.Join`, `fmt.Errorf` dengan `%w`.
- *Dependency injection lewat fungsi*: `Load` tidak memanggil `os.Getenv` langsung, jadi test tidak perlu mengubah env proses.
- Table-driven test + `t.Run`.
- `-include .env` (pakai tanda minus) artinya "boleh tidak ada". `.env` menimpa `.env.example` dan tidak ikut di-commit.

Kunci jawaban: `../kelana-example/internal/platform/config/`.

---

## Langkah 5 · Package `logx` (slog)

Buat `internal/platform/logx/logx.go` dengan tiga fungsi:

```go
// New: format "text" → slog.NewTextHandler, selain itu JSON.
func New(w io.Writer, level slog.Leveler, format string) *slog.Logger

// Menitipkan dan mengambil logger dari context.
type ctxKey struct{}
func WithLogger(ctx context.Context, l *slog.Logger) context.Context
func FromContext(ctx context.Context) *slog.Logger // fallback slog.Default()
```

Di `main.go`:

```go
logger := logx.New(os.Stdout, cfg.LogLevel, cfg.LogFormat)
slog.SetDefault(logger)
logger.Info("config loaded", "config", cfg)
```

Supaya `cfg` tercetak rapi, beri `Config` method `LogValue() slog.Value` (interface `slog.LogValuer`) yang mengembalikan `slog.GroupValue(...)`. Di Langkah 10 method ini juga menyamarkan password database.

**Cek**: `make run-api` mencetak log berformat `key=value`; ubah `KELANA_LOG_FORMAT=json` dan lihat bedanya.

**Konsep**: structured logging, key context bertipe sendiri (`struct{}`) supaya tidak bentrok, interface kecil `slog.LogValuer`.

---

## Langkah 6 · Fungsi `run()` dan graceful shutdown

Pecah `main` supaya isinya bisa dites:

```go
func main() {
	if err := run(context.Background(), os.LookupEnv, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "kelana-api:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, lookupEnv config.LookupFunc, stdout io.Writer) error {
	// 1. config.Load, logx.New
	// 2. ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM); defer stop()
	// 3. srv := &http.Server{Handler: mux, ReadHeaderTimeout: 5*time.Second, ...}
	// 4. ln, err := (&net.ListenConfig{}).Listen(ctx, "tcp", cfg.HTTPAddr)
	// 5. go func() { serveErr <- srv.Serve(ln) }()
	// 6. select { case err := <-serveErr: ... ; case <-ctx.Done(): }
	// 7. stop(); shutdownCtx dengan context.WithTimeout(context.WithoutCancel(ctx), cfg.ShutdownTimeout)
	// 8. srv.Shutdown(shutdownCtx), lalu tunggu <-serveErr == http.ErrServerClosed
}
```

Kenapa urutannya begitu:

- **Listen dulu, baru Serve**: kalau port sudah dipakai, error muncul sebelum log "server listening".
- **Channel `serveErr` ber-buffer 1**: goroutine tidak pernah macet walau tidak ada yang membaca.
- **`stop()` sebelum shutdown**: Ctrl+C kedua langsung mematikan proses.
- **`context.WithoutCancel`**: `ctx` sudah batal, jadi Shutdown butuh context baru yang punya batas waktu sendiri.
- **Timeout di `http.Server`**: tanpa `ReadHeaderTimeout`, klien lambat bisa menahan koneksi selamanya (gosec akan menegurmu di Langkah 12).

**Cek**

```sh
make run-api
# Ctrl+C → log "shutting down" lalu "server stopped", exit 0, make tidak melapor Error
```

Coba juga tambah handler sementara `GET /slow` yang `time.Sleep(5 * time.Second)`, panggil dengan curl, lalu Ctrl+C selagi request berjalan: request tetap selesai. Hapus handler itu setelahnya.

**Konsep**: goroutine + channel + `select`, `context` dan pembatalan, `signal.NotifyContext`. Bacaan: [Go Concurrency Patterns: Context](https://go.dev/blog/context).

Kunci jawaban: `../kelana-example/cmd/api/main.go`.

---

## Langkah 7 · `httpx`: helper respons JSON dan problem+json

Buat `internal/platform/httpx/response.go`:

- `WriteJSON(w, r, status, v)`: `json.Marshal` **dulu** ke memori; bila gagal kirim 500 (bukan respons yang terpotong).
- `type Problem struct { Type, Title, Status, Detail, Code, RequestID }` dengan struct tag JSON, sesuai [RFC 9457](https://www.rfc-editor.org/rfc/rfc9457) dan kode error di tab 05.
- `WriteProblem(w, r, p)`: Content-Type `application/problem+json`, `Type` default `about:blank`.

Lalu `health.go` dengan `Liveness()` yang memakai `WriteJSON`. Ganti handler inline di `main.go` dengan `httpx.Liveness()`.

**Cek**: `curl -i localhost:8080/healthz` tetap 200 dengan `Content-Type: application/json`.

**Konsep**: struct tag (`json:"detail,omitempty"`), `encoding/json`, kenapa error API butuh `code` yang stabil.

---

## Langkah 8 · Middleware dan `routes.go`

Di `internal/platform/httpx/middleware.go`:

```go
type Middleware func(http.Handler) http.Handler

// Chain(h, a, b) == a(b(h)): yang pertama berjalan paling dulu.
func Chain(h http.Handler, mws ...Middleware) http.Handler
```

Tiga middleware:

1. **`RequestID(base *slog.Logger)`**: pakai header `X-Request-Id` dari klien bila aman (1–128 karakter `[A-Za-z0-9._-]`), selain itu `rand.Text()` dari `crypto/rand`. Simpan di context, kirim balik di header respons, dan titipkan `base.With("request_id", id)` lewat `logx.WithLogger`.
2. **`AccessLog()`**: bungkus `ResponseWriter` dengan `statusRecorder` (mencatat status dan byte), lalu satu baris log per request. Status ≥ 500 → level Error. Beri method `Unwrap()` supaya `http.ResponseController` (Flush) tetap bekerja.
3. **`Recoverer()`**: `defer` + `recover()`, log stack trace, balas `WriteProblem` 500 `INTERNAL_ERROR`. Biarkan `http.ErrAbortHandler` tetap panic.

Pindahkan perakitan route ke `cmd/api/routes.go`:

```go
func routes(logger *slog.Logger) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /healthz", httpx.Liveness())
	return httpx.Chain(mux,
		httpx.RequestID(logger),
		httpx.AccessLog(),
		httpx.Recoverer(),
	)
}
```

**Cek**

```sh
curl -i localhost:8080/healthz                       # ada header X-Request-Id
curl -i -H 'X-Request-Id: abc-123' localhost:8080/healthz   # dipakai ulang
# log terminal: "http request" dengan request_id, method, path, status, duration_ms
```

Uji Recoverer dengan route sementara yang `panic("boom")`: harus 500 problem+json dan server tetap hidup. Hapus setelahnya.

**Konsep**: middleware `func(http.Handler) http.Handler`, closure, embedding struct (`statusRecorder` menanam `http.ResponseWriter`), `defer`/`recover`.

Kunci jawaban: `../kelana-example/internal/platform/httpx/middleware.go`, `../kelana-example/cmd/api/routes.go`.

---

## Langkah 9 · PostgreSQL dengan Docker Compose

`deploy/compose.yaml`:

```yaml
name: kelana

services:
  postgres:
    image: postgres:18
    environment:
      POSTGRES_USER: kelana
      POSTGRES_PASSWORD: kelana
      POSTGRES_DB: kelana
    ports:
      - "127.0.0.1:5432:5432"   # hanya dari laptop ini
    volumes:
      - pgdata:/var/lib/postgresql   # PG 18: bukan .../data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U kelana -d kelana"]
      interval: 5s
      timeout: 3s
      retries: 10
      start_period: 10s

volumes:
  pgdata:
```

Tambahkan ke Makefile:

```makefile
COMPOSE := docker compose -f deploy/compose.yaml

up: ## Nyalakan PostgreSQL dan tunggu sampai sehat
	$(COMPOSE) up -d --wait

down: ## Matikan container (data tetap di volume)
	$(COMPOSE) down

logs: ## Ikuti log container
	$(COMPOSE) logs -f
```

(Jangan lupa tambahkan `up down logs` ke `.PHONY`.)

**Cek**

```sh
make up
docker compose -f deploy/compose.yaml ps     # STATUS: healthy
docker compose -f deploy/compose.yaml exec postgres psql -U kelana -c 'select version();'
```

Bila port 5432 sudah dipakai PostgreSQL lain, ganti port kiri menjadi `127.0.0.1:5433:5432`.

---

## Langkah 10 · Package `database` dan `/readyz`

1. Tambah `KELANA_DATABASE_URL=postgres://kelana:kelana@localhost:5432/kelana?sslmode=disable` ke `.env.example`.
2. `internal/platform/database/database.go`:
   - `Addr(databaseURL) (string, error)`: parse URL, wajib skema `postgres`/`postgresql`, ambil `host:port` (default port 5432). **Pesan error tidak boleh memuat URL**, karena URL berisi password (ambil `urlErr.Err` lewat `errors.As`).
   - `PingTCP(databaseURL) (func(context.Context) error, error)`: mengembalikan fungsi yang `net.Dialer.DialContext` ke alamat itu lalu menutupnya.
3. `config`: tambah `DatabaseURL` (wajib), validasi dengan `database.Addr`, dan samarkan di `LogValue` dengan `url.Redacted()`.
4. `httpx/health.go`: `type Check func(ctx context.Context) error` dan `Readiness(timeout, map[string]Check)`. Jalankan setiap check dengan `context.WithTimeout`; semua lolos → 200, satu gagal → 503. Detail error **hanya ke log**, klien cukup melihat `"unavailable"`.
5. `routes(logger, dbCheck)` mendaftarkan `GET /readyz`.

**Cek**

```sh
make up && make run-api
curl -i localhost:8080/readyz     # 200 {"status":"ok","checks":{"database":"ok"}}
make down
curl -i localhost:8080/readyz     # 503, sedangkan /healthz tetap 200
```

**Konsep**

- Liveness vs readiness: database mati bukan alasan me-restart API (liveness tetap 200), tetapi alasan berhenti menerima traffic (readiness 503).
- Fungsi sebagai nilai (`Check`), closure yang menangkap `addr`.
- Di Fase 1, `PingTCP` diganti `pool.Ping` dari pgx yang benar-benar login.

---

## Langkah 11 · Melengkapi test

Target: setiap package di `internal/platform` dan `cmd/api` punya test, dan `make test` hijau.

| File test | Yang dites | Alat |
| --- | --- | --- |
| `config_test.go` | default, override, kumpulan error, password tersamar | table-driven |
| `logx_test.go` | format JSON/text, level, context round-trip | `bytes.Buffer` |
| `database_test.go` | `Addr` berbagai URL, error tanpa password, `PingTCP` | `net.Listen("tcp", "127.0.0.1:0")` |
| `response_test.go`, `health_test.go`, `middleware_test.go` | status, header, body, urutan Chain, panic | `httptest.NewRecorder`, `httptest.NewRequest` |
| `cmd/api/main_test.go` | `run()` melayani `/healthz` lalu berhenti saat ctx dibatalkan; error config; port terpakai | env palsu + `context.WithCancel` |

Kunci jawaban ada di setiap `*_test.go` di `../kelana-example`. Coba tulis dulu dari daftar nama test ini (`grep -n '^func Test' -r ../kelana-example`), baru bandingkan.

**Cek**

```sh
make test
go test -cover ./...
```

**Konsep**: `httptest`, port `:0` untuk port acak, kenapa `run()` menerima `ctx`/`lookupEnv`/`stdout` sebagai parameter (supaya bisa dites tanpa `os.Exit` dan tanpa env asli).

---

## Langkah 12 · Linter dan govulncheck

Buat `.golangci.yml` (format v2). Isi pentingnya:

- `linters.default: standard` + `bodyclose`, `depguard`, `errorlint`, `gosec`, `noctx`, `revive`, `unconvert`.
- Aturan `depguard`: file di `**/internal/platform/**` dilarang meng-import `kelana/internal/{identity,catalog,search,supplier,inventory,booking,payment,notification}`. Ini aturan dependensi tab 07 yang dijaga otomatis.
- `formatters`: `gofmt`, `goimports` dengan `local-prefixes: [kelana]`.

Tambahkan ke Makefile:

```makefile
lint: ## golangci-lint lalu govulncheck
	golangci-lint run ./...
	$(MAKE) vuln

vuln: ## Cek kerentanan stdlib dan dependency
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...
```

**Cek**

```sh
golangci-lint config verify
make lint
```

Perbaiki semua temuan sendiri. revive biasanya meminta komentar dokumentasi untuk setiap identifier yang diekspor, diawali namanya.

Kunci jawaban: `../kelana-example/.golangci.yml`.

---

## Langkah 13 · CI di GitHub Actions

`.github/workflows/ci.yml` dengan dua job:

1. **test**: `actions/checkout@v6`, `actions/setup-go@v6` (`go-version-file: go.mod`, `cache: false` karena belum ada `go.sum`), lalu `go mod tidy -diff`, `go vet ./...`, `go test -race -count=1 ./...`, govulncheck.
2. **lint**: `golangci/golangci-lint-action@v9` dengan `version: v2.13`.

Pasang `permissions: contents: read` di level workflow.

**Cek**

```sh
gh repo create kelana --private --source=. --push   # atau buat repo lewat web lalu git push
gh run watch
```

Bila ingin module path GitHub (`github.com/USERNAME/kelana`), ganti **sekarang** selagi kodenya sedikit. Caranya ada di bagian "Push ke GitHub" pada `../kelana-example/README.md`.

---

## Langkah 14 · Playground latihan

Playground adalah module Go **terpisah**, supaya `make test` dan CI di root tidak menjalankannya.

```sh
mkdir -p playground/{bookingcode,pricing}
cd playground && go mod init kelana/playground
```

Salin **hanya file test** dan kerangka fungsi dari `../kelana-example/playground/` (ini satu-satunya bagian yang boleh disalin, karena test-nya adalah soal), lalu isi TODO-nya sendiri:

- `bookingcode.Generate()`: 6 karakter dari alfabet 32 karakter (BR-10), pakai `crypto/rand`.
- `pricing.Calculate(Pax, Fare)`: aturan BR-02, BR-04, BR-05, uang `int64`, kalikan dulu baru bagi.

**Cek**

```sh
cd playground && go test -race ./...
```

Setelah hijau, tambahkan minimal satu kasus test buatanmu sendiri di `pricing_test.go`.

---

## Langkah 15 · README, docs, dan checklist Fase 0

- `README.md`: prasyarat, cara mulai, tabel env var, daftar perintah, struktur folder, dan hal yang sengaja belum ada.
- `docs/README.md`: tautan ke dokumen desain di Claude.
- Tag rilis setelah semua hijau: `git tag v0.0.1`.

Checklist Fase 0 (tab 09, "Selesai bila"):

- [ ] Latihan di `playground/` hijau
- [ ] `make up && make run-api`, lalu `curl localhost:8080/healthz` menjawab 200
- [ ] Ctrl+C mematikan server dengan rapi
- [ ] `make test` dan `make lint` hijau
- [ ] Repo di-push ke GitHub dan CI hijau
- [ ] Kamu bisa menjelaskan setiap file tanpa membukanya

Hapus atau pindahkan file panduan ini ke `docs/` setelah selesai.

---

## Setelah Fase 0

Fase 1 (identity & catalog) menambahkan dependency pertama: pgx, sqlc, goose. Saat itu:

- `go.sum` muncul → hapus `cache: false` di CI.
- Pin goose dan govulncheck dengan `go get -tool ...` lalu jalankan via `go tool ...` (tab 07).
- Tambah `db/migrations/`, `db/queries/`, `sqlc.yaml`, dan target `migrate-up`, `migrate-down`, `migrate-new`, `sqlc` di Makefile.
