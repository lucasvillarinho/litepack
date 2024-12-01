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
	@go test -v -coverprofile=rawcover.out -json $(filter-out \
		$(shell go list ./... | grep -E "github.com/lucasvillarinho/litepack/internal/log/queries|github.com/lucasvillarinho/litepack/internal/cron/mocks|github.com/lucasvillarinho/litepack/cache/queries|github.com/lucasvillarinho/litepack/database/mocks|github.com/lucasvillarinho/litepack/internal/log/mocks"), \
		$(shell go list ./...)) 2>&1 | tee /tmp/gotest.log | gotestfmt -hide successful-tests,empty-packages

.PHONY: gen-sqlc-cache
gen-sqlc-cache:
	@echo "Generating sqlc cache..."
	@sqlc generate -f cache/configs/sqlc.yaml
	@echo "sqlc cache generated successfully"

.PHONY: gen-mocks-database
gen-mocks-database:
	@echo "Generating mocks with mockery..."
	@mockery --config database/configs/.mockery.yaml
	@echo "Mocks generated successfully"

.PHONY: gen-sqlc-log
gen-sqlc-log:
	@echo "Generating sqlc cache..."
	@sqlc generate -f internal/log/configs/sqlc.yaml
	@echo "sqlc log generated successfully"

.PHONY: gen-mocks-log
gen-mocks-log:
	@echo "Generating mocks with mockery..."
	@mockery --config internal/log/configs/.mockery.yaml
	@echo "Mocks generated successfully"

.PHONY: gen-mocks-cron
gen-mocks-cron:
	@echo "Generating mocks with mockery..."
	@mockery --config internal/cron/configs/.mockery.yaml
	@echo "Mocks generated successfully"
