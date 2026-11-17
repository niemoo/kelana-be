# Semua perintah harian Kelana. Jalankan `make` untuk melihat daftarnya.

# Nilai default untuk development lokal ada di .env.example; .env (bila ada)
# menimpanya. Keduanya diekspor sebagai environment variable ke setiap perintah.
include .env.example
-include .env
export

COMPOSE := docker compose -f deploy/compose.yaml

.DEFAULT_GOAL := help
.PHONY: help up down logs run-api build test lint vuln tidy clean

help: ## Tampilkan daftar target
	@awk 'BEGIN {FS = ":.*## "} /^[a-zA-Z_-]+:.*## / {printf "  %-10s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

up: ## Nyalakan PostgreSQL di Docker dan tunggu sampai sehat
	$(COMPOSE) up -d --wait

down: ## Matikan container (data tetap tersimpan di volume)
	$(COMPOSE) down

logs: ## Ikuti log container
	$(COMPOSE) logs -f

# Sengaja tidak memakai `go run`: go run keluar dengan status 1 setiap kali
# menerima Ctrl+C, sehingga make melaporkan "Error 1" walau shutdown-nya bersih.
run-api: build ## Build lalu jalankan API (Ctrl+C untuk berhenti)
	./bin/kelana-api

build: ## Build binary ke bin/kelana-api
	go build -o bin/kelana-api ./cmd/api

test: ## Unit test dengan race detector
	go test -race -count=1 ./...

lint: ## golangci-lint lalu govulncheck
	golangci-lint run ./...
	$(MAKE) vuln

vuln: ## Cek kerentanan yang diketahui di stdlib dan dependency
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

tidy: ## Rapikan go.mod dan go.sum
	go mod tidy

clean: ## Hapus hasil build
	rm -rf bin
