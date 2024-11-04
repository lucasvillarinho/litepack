.PHONY: build
build:
	@echo "Building..."
	@go build -o bin/ ./...
	@echo "Build completed successfully"

.PHONY: format
format: ## Format code
	@echo "Formatting code..."
	@gofumpt -w .
	@goimports -w .
	@golines -m 100 -w .
	@fieldalignment -fix ./...
	@echo "Code formatted successfully"

.PHONY: lint
lint: build ## Run lint
	@echo "Running linter..."
	@golangci-lint run ./...
	@echo "Linter passed successfully"