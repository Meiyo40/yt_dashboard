.DEFAULT_GOAL := help

APP_NAME := yt-dashboard
BIN_DIR := bin
BIN := $(BIN_DIR)/$(APP_NAME).exe

.PHONY: help
help: ## Show available commands
	@echo YouTube Comment Dashboard commands:
	@echo.
	@echo   make setup       Install development tools and download Go modules
	@echo   make deps        Download Go modules
	@echo   make tidy        Tidy Go modules
	@echo   make generate    Generate templ Go files
	@echo   make fmt         Format Go code
	@echo   make vet         Run go vet
	@echo   make test        Generate templates and run tests
	@echo   make check       Run fmt, generate, vet, and tests
	@echo   make run         Generate templates and run the app
	@echo   make dev         Start Air hot reload server
	@echo   make build       Generate templates and build production binary
	@echo   make clean       Remove build and hot-reload artifacts
	@echo   make env         Create .env from env.example if missing

.PHONY: setup
setup: ## Install development tools and download Go modules
	go install github.com/a-h/templ/cmd/templ@latest
	go install github.com/air-verse/air@latest
	go mod download

.PHONY: deps
deps: ## Download Go modules
	go mod download

.PHONY: tidy
tidy: ## Tidy Go modules
	go mod tidy

.PHONY: generate
generate: ## Generate templ Go files
	templ generate

.PHONY: fmt
fmt: ## Format Go code
	go fmt ./...

.PHONY: vet
vet: ## Run go vet
	go vet ./...

.PHONY: test
test: generate ## Generate templates and run tests
	go test ./...

.PHONY: check
check: fmt generate vet test ## Run formatting, generation, vetting, and tests

.PHONY: run
run: generate ## Generate templates and run the app
	go run .

.PHONY: dev
dev: ## Start Air hot reload server
	air

.PHONY: build
build: generate ## Generate templates and build production binary
	powershell -NoProfile -Command "New-Item -ItemType Directory -Force -Path '$(BIN_DIR)' | Out-Null"
	go build -o $(BIN) .

.PHONY: clean
clean: ## Remove build and hot-reload artifacts
	powershell -NoProfile -Command "Remove-Item -Recurse -Force 'tmp' -ErrorAction SilentlyContinue; Remove-Item -Force '$(BIN)' -ErrorAction SilentlyContinue"

.PHONY: env
env: ## Create .env from env.example if missing
	powershell -NoProfile -Command "if (!(Test-Path '.env') -and (Test-Path 'env.example')) { Copy-Item 'env.example' '.env' }"
