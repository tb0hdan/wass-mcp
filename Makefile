.PHONY: all build tools tag test test-integration
VERSION ?= $(shell cat cmd/wass-mcp/VERSION)
LINTER_VERSION ?= v2.8.0

all: lint test build

lint:
	@echo "Running linters..."
	@golangci-lint run ./...

build:
	@echo "Building the project..."
	@go build -o build/wass-mcp ./cmd/wass-mcp/*.go

build-dir:
	@if [ ! -d build/ ]; then mkdir -p build; fi

tools:
	@echo "Running tools..."
	@curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b $(shell go env GOPATH)/bin $(LINTER_VERSION)

tag:
	@echo "Tagging the current version..."
	git tag -a "v$(VERSION)" -m "Release version $(VERSION)"; \
	git push origin "v$(VERSION)"

test: build-dir
	@echo "Running tests with coverage..."
	@go test -v -race -coverprofile=build/coverage.out ./...
	@go tool cover -html=build/coverage.out -o build/coverage.html

docker-build:
	@echo "Building Docker image..."
	@docker build -t tb0hdan/wass-mcp -f deployments/Dockerfile .

docker-run:
	@echo "Running Docker container..."
	@docker run -p 8989:8989 -v wass-data:/data tb0hdan/wass-mcp /app/wass-mcp --bind 0.0.0.0:8989 --db /data/wass-mcp.db --debug

docker-tag: docker-build
	@echo "Tagging Docker image..."
	@docker tag tb0hdan/wass-mcp tb0hdan/wass-mcp:$(VERSION)
	@docker tag tb0hdan/wass-mcp tb0hdan/wass-mcp:latest
	@docker push tb0hdan/wass-mcp:$(VERSION)
	@docker push tb0hdan/wass-mcp:latest
