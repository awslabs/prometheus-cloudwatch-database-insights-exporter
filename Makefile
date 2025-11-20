include ${BGO_MAKEFILE}

.PHONY: format lint test check clean help

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
check: format lint coverage
	@echo "All checks passed!"

# Clean build artifacts
dbi-clean:
	@echo "Cleaning build artifacts..."
	rm -rf build/
	rm -f coverage.out coverage.html
	rm -f dbinsights-exporter

# Show available targets
dbi-help:
	@echo "Available targets:"
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
