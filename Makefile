.PHONY: build run format lint test check clean help

# Format code using go fmt
format:
	@echo "Formatting Go code..."
	go fmt ./...

# Run static analysis using go vet
lint:
	@echo "Running Go vet..."
	go vet ./...

# Run tests with coverage
coverage:
	@echo "Running tests with coverage..."
	go test -cover ./...

coverage-profile:
	@echo "Generating coverage profile..."
	go test -coverprofile=coverage.out ./...
	@echo "Coverage profile saved to coverage.out"

coverage-html: coverage-profile
	@echo "Generating HTML coverage report..."
	go tool cover -html=coverage.out -o coverage.html
	@echo "Open coverage.html in your browser"

# Run basic checks
check: build format lint coverage-profile
	@echo "All checks passed!"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf build/
	rm -f coverage.out coverage.html
	rm -f dbinsights-exporter

# Build
build:
	@echo "Building project to dbinsights-exporter..."
	go build -o dbinsights-exporter ./cmd

release: check
	@echo "Building release target of project to dbinsights-exporter..."

# Build and run project
run: build
	@echo "Running database insights exporter..."
	./dbinsights-exporter

# Show available targets
help:
	@echo "Available targets:"
	@echo "  build            - Build project"
	@echo "  run              - Build and Run project"
	@echo "  format           - Format Go code"
	@echo "  lint             - Static analysis with go vet"
	@echo "  check            - Format, lint, and test"
	@echo "  clean            - Clean build artifacts"
	@echo ""
	@echo "Coverage:"
	@echo "  coverage         - Run all tests with coverage report"
	@echo "  coverage-html    - Interactive HTML report"
	@echo ""
	@echo "  help             - Show this help"
