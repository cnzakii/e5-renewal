.PHONY: ui build dist check lint fmt test bench fuzz clean
VERSION ?= dev
GOLANGCI_LINT ?= golangci-lint
GORELEASER ?= goreleaser
GO_TOOLCHAIN := $(shell sed -n 's/^toolchain //p' go.mod)
GO_PACKAGES := ./cmd/... ./internal/... ./web/...

ui:
	cd frontend && bun install --frozen-lockfile
	cd frontend && bun run build
	mkdir -p web/dist
	find web/dist -mindepth 1 ! -name .gitkeep -delete
	cp -R frontend/dist/. web/dist/

build: ui
	CGO_ENABLED=0 go build -trimpath -buildvcs=false -ldflags="-s -w -X main.version=$(VERSION)" -o bin/e5-renewal ./cmd/e5-renewal

dist: ui
	GOTOOLCHAIN=$(GO_TOOLCHAIN) BUILD_VERSION="$(VERSION)" $(GORELEASER) release --snapshot --clean

fmt:
	$(GOLANGCI_LINT) fmt $(GO_PACKAGES)
	$(GOLANGCI_LINT) run --fix --enable-only=wsl_v5 $(GO_PACKAGES)
	cd frontend && bun run lint --fix
	cd frontend && bun run format

lint:
	$(GOLANGCI_LINT) run $(GO_PACKAGES)
	cd frontend && bun run format:check
	cd frontend && bun run lint

test:
	go test -race $(GO_PACKAGES)
	cd frontend && bun run test

check: lint test
	cd frontend && bun run build

bench:
	go test -run '^$$' -bench . -benchmem ./internal/app

fuzz:
	go test -run '^$$' -fuzz FuzzAccountValidation -fuzztime 10s -parallel 2 ./internal/app

clean:
	rm -rf bin dist
	find web/dist -mindepth 1 ! -name .gitkeep -delete
