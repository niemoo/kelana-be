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