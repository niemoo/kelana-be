# Kelana

Prototype backend REST API pemesanan tiket pesawat, dibangun dengan Go untuk belajar. Repo ini berada di **Fase 0**: fondasi repo, health check, konfigurasi dari env, logging `slog`, graceful shutdown, dan CI.

Dokumen desain (PRD, desain database, spesifikasi API, ADR, roadmap) ada di [docs/](docs/README.md).

## Prasyarat

- Go 1.27 (`go version`)
- Docker dengan Compose v2 (`docker compose version`)
- make
- golangci-lint v2.13 atau lebih baru untuk `make lint`, karena versi sebelumnya belum mengenal Go 1.27 ([cara pasang](https://golangci-lint.run/welcome/install/))
- Linux, macOS, atau WSL2 di Windows; race detector di `make test` butuh cgo

## Mulai

```sh
make up        # PostgreSQL 18 di Docker, menunggu sampai sehat
make run-api   # build, lalu API jalan di http://localhost:8080
```

Di terminal lain:

```sh
curl -i localhost:8080/healthz   # 200 {"status":"ok"}
curl -i localhost:8080/readyz    # 200 bila PostgreSQL hidup, 503 bila mati
```

Tekan Ctrl+C di terminal `make run-api`. Log menampilkan `shutting down` lalu `server stopped`: request yang sedang berjalan diberi waktu selesai sampai `KELANA_SHUTDOWN_TIMEOUT`. Ctrl+C kedua mematikan proses seketika.

Coba juga `make down` saat API hidup: `/readyz` berubah menjadi 503, sedangkan `/healthz` tetap 200.

Bila `make up` gagal karena port 5432 sudah dipakai (misalnya ada PostgreSQL lain di laptop), matikan dulu yang lain itu, atau ganti port kiri di `deploy/compose.yaml` (mis. `127.0.0.1:5433:5432`) beserta port di `KELANA_DATABASE_URL` pada `.env`.

## Konfigurasi

Semua konfigurasi berasal dari environment variable berprefix `KELANA_` dan divalidasi saat start. Bila ada yang salah, API berhenti sambil menyebutkan semua kesalahannya sekaligus.

Makefile membaca nilai dari `.env.example`. Untuk mengubahnya, jalankan `cp .env.example .env` lalu edit `.env`; file itu tidak ikut di-commit.

| Variabel | Default di kode | Keterangan |
| --- | --- | --- |
| `KELANA_HTTP_ADDR` | `:8080` | Alamat server HTTP |
| `KELANA_DATABASE_URL` | wajib diisi | mis. `postgres://kelana:kelana@localhost:5432/kelana?sslmode=disable` |
| `KELANA_LOG_LEVEL` | `info` | `debug`, `info`, `warn`, atau `error` |
| `KELANA_LOG_FORMAT` | `json` | `text` lebih enak dibaca saat lokal |
| `KELANA_SHUTDOWN_TIMEOUT` | `10s` | Batas waktu graceful shutdown |

`.env.example` sengaja memakai `debug` dan `text` supaya log lokal mudah dibaca.

## Perintah

| Perintah | Fungsi |
| --- | --- |
| `make up` / `make down` | Nyalakan / matikan PostgreSQL |
| `make run-api` | Build lalu jalankan API |
| `make test` | Unit test dengan race detector |
| `make lint` | golangci-lint lalu govulncheck |
| `make build` | Build ke `bin/kelana-api` |

`make` tanpa argumen menampilkan daftar lengkap.

## Struktur

```text
cmd/api/                  main.go (run + graceful shutdown), routes.go
internal/platform/
  config/                 baca dan validasi env KELANA_*
  database/               cek koneksi PostgreSQL; pool pgx menyusul di Fase 1
  httpx/                  middleware, JSON & problem+json, /healthz dan /readyz
  logx/                   setup slog, logger di context
deploy/compose.yaml       PostgreSQL 18
playground/               latihan Fase 0 (module terpisah, tidak ikut CI)
.github/workflows/ci.yml  vet, test -race, govulncheck, golangci-lint
```

Di Fase 0 baru ada `platform`; modul bisnis (`identity`, `catalog`, dan seterusnya) menyusul sesuai roadmap. Aturan dependensinya sudah dijaga depguard di `.golangci.yml`: package di `internal/platform` tidak boleh meng-import modul bisnis.

## Checklist Fase 0

- [ ] Latihan di `playground/` hijau (lihat [playground/README.md](playground/README.md))
- [ ] `make up && make run-api`, lalu `curl localhost:8080/healthz` menjawab 200
- [ ] Ctrl+C mematikan server dengan rapi
- [ ] `make test` dan `make lint` hijau
- [ ] Repo di-push ke GitHub dan CI hijau

## Yang sengaja belum ada

- `/readyz` baru memastikan port PostgreSQL terbuka lewat koneksi TCP. Di Fase 1, check ini diganti `pool.Ping` dari pgx yang benar-benar login.
- Request ID dibuat dengan `crypto/rand.Text()`. Bila ingin UUIDv7 seperti contoh di tab 05, ganti saat ID entitas mulai dibuat di Fase 1.
- `go.mod` belum punya dependency, jadi belum ada `go.sum`. Setelah dependency pertama masuk, hapus `cache: false` di CI, lalu pin goose dan govulncheck dengan `go get -tool` sesuai tab 07.

## Push ke GitHub

Module path saat ini `kelana`. Bila ingin memakai path GitHub, ganti sekarang selagi kodenya masih sedikit:

```sh
go mod edit -module github.com/USERNAME/kelana
grep -rln 'kelana/internal' --include='*.go' --include='.golangci.yml' . \
  | xargs sed -i.bak 's#kelana/internal#github.com/USERNAME/kelana/internal#g'
sed -i.bak 's#^        - kelana$#        - github.com/USERNAME/kelana#' .golangci.yml
find . -name '*.bak' -delete
go build ./... && make test
```
