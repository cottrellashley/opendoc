.PHONY: help build build-fast install serve workbench test lint clean \
       docker-build docker-up docker-down docker-shell docker-logs

BINARY := opendoc
GO_FLAGS := -ldflags="-s -w"

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-18s\033[0m %s\n", $$1, $$2}'

build: ## Build optimised binary
	CGO_ENABLED=0 go build $(GO_FLAGS) -o $(BINARY) ./cmd/opendoc

build-fast: ## Build without optimisations
	go build -o $(BINARY) ./cmd/opendoc

install: build ## Install to $GOPATH/bin
	go install ./cmd/opendoc

serve: build ## Serve example site locally
	./$(BINARY) serve example -p 8000

workbench: build ## Start workbench for example site
	./$(BINARY) workbench example -p 3000

test: ## Run tests
	go test ./...

lint: ## Run go vet
	go vet ./...

docker-build: ## Build Docker image
	docker compose build

docker-up: ## Start container (localhost:3000)
	docker compose up -d

docker-down: ## Stop container
	docker compose down

docker-shell: ## Shell into container
	docker compose exec opendoc bash

docker-logs: ## Tail container logs
	docker compose logs -f

clean: ## Remove build artifacts
	rm -f $(BINARY)
	rm -rf dist dist-publish
