.PHONY: all build test lint format clean release-snapshot install help

# Show available targets
help:
	@echo "merklediff - Makefile targets"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Development:"
	@echo "  build            Build for current platform"
	@echo "  run              Run the CLI"
	@echo "  test             Run tests with race detection"
	@echo "  test-coverage    Run tests with coverage report"
	@echo "  lint             Run go vet"
	@echo "  format           Format code with go fmt and goimports"
	@echo "  lc               Count lines of Go code"
	@echo ""
	@echo "Setup:"
	@echo "  deps             Download and tidy dependencies"
	@echo "  dev-setup        Install development tools"
	@echo "  install          Install to GOPATH/bin"
	@echo ""
	@echo "Cross-compilation:"
	@echo "  build-linux      Build for Linux (amd64, arm64)"
	@echo "  build-darwin     Build for macOS (amd64, arm64)"
	@echo "  build-windows    Build for Windows (amd64)"
	@echo "  build-all        Build for all platforms"
	@echo ""
	@echo "Release:"
	@echo "  release-snapshot Test GoReleaser locally"
	@echo "  clean            Remove build artifacts"
	@echo ""
	@echo "Shortcuts:"
	@echo "  all              Run lint, test, and build (default)"

# Default target
all: lint test build

# Build for current platform
build:
	go build -o ./build/merklediff ./cmd/merklediff

# Install to GOPATH/bin
install:
	go install ./cmd/merklediff

# Run tests
test:
	go test -v -race ./...

# Run tests with coverage
test-coverage:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Lint
lint:
	go vet ./...

# Format
format:
	go fmt ./...
	goimports -w .

# Clean build artifacts
clean:
	rm -rf ./build ./dist coverage.out coverage.html

# Cross-compile builds
build-linux:
	GOOS=linux GOARCH=amd64 go build -o ./build/merklediff-linux-amd64 ./cmd/merklediff
	GOOS=linux GOARCH=arm64 go build -o ./build/merklediff-linux-arm64 ./cmd/merklediff

build-darwin:
	GOOS=darwin GOARCH=amd64 go build -o ./build/merklediff-darwin-amd64 ./cmd/merklediff
	GOOS=darwin GOARCH=arm64 go build -o ./build/merklediff-darwin-arm64 ./cmd/merklediff

build-windows:
	GOOS=windows GOARCH=amd64 go build -o ./build/merklediff-windows-amd64.exe ./cmd/merklediff

build-all: build-linux build-darwin build-windows

# GoReleaser snapshot (local test)
release-snapshot:
	goreleaser release --snapshot --clean

# Run the CLI
run:
	go run ./cmd/merklediff

# Line count
lc:
	@find . -name "*.go" -not -path "./vendor/*" | xargs wc -l | tail -1

# Dependencies
deps:
	go mod download
	go mod tidy

# Dev setup (install tools)
dev-setup:
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/goreleaser/goreleaser/v2@latest
