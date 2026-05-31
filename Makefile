WEB_DIR := apps/web
GO ?= go
NPM ?= npm
GOHOSTOS := $(shell $(GO) env GOHOSTOS)
GOHOSTARCH := $(shell $(GO) env GOHOSTARCH)
GO_CACHE_DIR := $(CURDIR)/.cache/go-build

.PHONY: setup dev dev-api dev-web docker-up docker-down package build test fmt clean

setup:
	cd $(WEB_DIR) && $(NPM) ci

dev:
	@echo "Starting API and web dev servers. Press Ctrl+C to stop both."
	@$(MAKE) dev-api & api_pid=$$!; \
	$(MAKE) dev-web & web_pid=$$!; \
	trap 'kill $$api_pid $$web_pid 2>/dev/null' INT TERM EXIT; \
	wait

dev-api:
	GOCACHE=$(GO_CACHE_DIR) GOOS=$(GOHOSTOS) GOARCH=$(GOHOSTARCH) $(GO) run ./cmd/api

dev-web:
	cd $(WEB_DIR) && $(NPM) run dev

docker-up:
	docker compose up --build

docker-down:
	docker compose down

package:
	node scripts/package-release.mjs

build:
	GOCACHE=$(GO_CACHE_DIR) GOOS=$(GOHOSTOS) GOARCH=$(GOHOSTARCH) $(GO) build ./...
	cd $(WEB_DIR) && $(NPM) run build

test:
	node scripts/check-web-dialogs.mjs
	GOCACHE=$(GO_CACHE_DIR) GOOS=$(GOHOSTOS) GOARCH=$(GOHOSTARCH) $(GO) test ./...
	cd $(WEB_DIR) && $(NPM) run build

fmt:
	GOCACHE=$(GO_CACHE_DIR) $(GO) fmt ./...

clean:
	rm -rf data $(WEB_DIR)/.next
