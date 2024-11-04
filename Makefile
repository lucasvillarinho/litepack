.PHONY: format
format: ## Format code
	@echo "Formatting code..."
	@gofumpt -w .
	@goimports -w .
	@golines -m 100 -w .
	@fieldalignment -fix ./...
	@echo "Code formatted successfully"