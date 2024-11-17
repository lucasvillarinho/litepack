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

.PHONY: test
test:  ## Run tests
	@go test -v -coverprofile=rawcover.out -json $$(go list ./...) 2>&1 | tee /tmp/gotest.log | gotestfmt -hide successful-tests,empty-packages
	@go test


.PHONY: generate-sqlc-cache 
generate-sqlc-cache:
	@echo "Generating sqlc cache..."
	@sqlc generate -f cache/configs/sqlc.yaml
	@echo "sqlc cache generated successfully"