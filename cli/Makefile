# ── VibeCloud Makefile ────────────────────────────────────────────────────────

.PHONY: help build test lint clean

help: ## Show this help
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / \
	  {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Compile CLI binary → vibecloud
	go build -ldflags="-s -w" -o vibecloud .

test: ## Run tests
	go test ./...

lint: ## Run go vet
	go vet ./...

clean: ## Remove build artifacts
	rm -f vibecloud
