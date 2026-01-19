.PHONY: test fmt lint build ci clean install help

# Default target
.DEFAULT_GOAL := help

# Test runs all tests
test:
	go test ./...

# Fmt formats all Go code
fmt:
	gofmt -w .

# Lint runs golangci-lint if available, otherwise runs go vet
lint:
	@which golangci-lint > /dev/null 2>&1 && golangci-lint run || go vet ./...

# Build compiles the ralph binary
build:
	go build -o ralph ./cmd/ralph

# CI runs all checks that CI runs (test + format check)
ci: test
	@echo "Checking code formatting..."
	@gofmt -l . | grep -v "^$$" && echo "Code is not formatted. Run 'make fmt'" && exit 1 || echo "Code formatting OK"

# Clean removes build artifacts
clean:
	rm -f ralph

# Install builds and installs ralph to GOPATH/bin
install:
	go install ./cmd/ralph

# Help displays available targets
help:
	@echo "Available targets:"
	@echo "  make test      - Run all tests"
	@echo "  make fmt       - Format all Go code"
	@echo "  make lint      - Run linter (golangci-lint or go vet)"
	@echo "  make build     - Build the ralph binary"
	@echo "  make ci        - Run CI checks (tests + format check)"
	@echo "  make clean     - Remove build artifacts"
	@echo "  make install   - Install ralph to GOPATH/bin"
	@echo "  make help      - Show this help message"
