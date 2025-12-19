.PHONY: all setup build test lint clean docker-up docker-down help

# Colors for output
CYAN := \033[36m
GREEN := \033[32m
YELLOW := \033[33m
RESET := \033[0m

# Default target
all: lint test build

# Setup development environment
setup:
	@echo "$(CYAN)Setting up development environment...$(RESET)"
	@echo "$(YELLOW)Installing Go dependencies...$(RESET)"
	cd goreview && go mod download
	@echo "$(YELLOW)Installing Node.js dependencies...$(RESET)"
	cd integrations/github-app && pnpm install
	@echo "$(YELLOW)Installing development tools...$(RESET)"
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/air-verse/air@latest
	@echo "$(GREEN)Setup complete!$(RESET)"

# Build all projects
build:
	@echo "$(CYAN)Building all projects...$(RESET)"
	$(MAKE) -C goreview build
	cd integrations/github-app && pnpm build
	@echo "$(GREEN)Build complete!$(RESET)"

# Run all tests
test:
	@echo "$(CYAN)Running all tests...$(RESET)"
	$(MAKE) -C goreview test
	cd integrations/github-app && pnpm test
	@echo "$(GREEN)All tests passed!$(RESET)"

# Run linters
lint:
	@echo "$(CYAN)Running linters...$(RESET)"
	$(MAKE) -C goreview lint
	cd integrations/github-app && pnpm lint
	@echo "$(GREEN)Linting complete!$(RESET)"

# Clean all build artifacts
clean:
	@echo "$(CYAN)Cleaning build artifacts...$(RESET)"
	$(MAKE) -C goreview clean
	cd integrations/github-app && pnpm clean
	rm -rf coverage/
	@echo "$(GREEN)Clean complete!$(RESET)"

# Start Docker services
docker-up:
	@echo "$(CYAN)Starting Docker services...$(RESET)"
	docker compose up -d
	@echo "$(GREEN)Services started!$(RESET)"
	@echo "$(YELLOW)Ollama: http://localhost:11434$(RESET)"
	@echo "$(YELLOW)GitHub App: http://localhost:3000$(RESET)"

# Stop Docker services
docker-down:
	@echo "$(CYAN)Stopping Docker services...$(RESET)"
	docker compose down
	@echo "$(GREEN)Services stopped!$(RESET)"

# Pull Ollama model
ollama-pull:
	@echo "$(CYAN)Pulling Ollama model...$(RESET)"
	docker compose exec ollama ollama pull qwen2.5-coder:7b
	@echo "$(GREEN)Model ready!$(RESET)"

# Show logs
logs:
	docker compose logs -f

# Development mode - watch all projects
dev:
	@echo "$(CYAN)Starting development mode...$(RESET)"
	@echo "$(YELLOW)Run in separate terminals:$(RESET)"
	@echo "  cd goreview && make dev"
	@echo "  cd integrations/github-app && pnpm dev"

# Help
help:
	@echo "$(CYAN)Available targets:$(RESET)"
	@echo "  $(GREEN)setup$(RESET)        - Setup development environment"
	@echo "  $(GREEN)build$(RESET)        - Build all projects"
	@echo "  $(GREEN)test$(RESET)         - Run all tests"
	@echo "  $(GREEN)lint$(RESET)         - Run linters"
	@echo "  $(GREEN)clean$(RESET)        - Clean build artifacts"
	@echo "  $(GREEN)docker-up$(RESET)    - Start Docker services"
	@echo "  $(GREEN)docker-down$(RESET)  - Stop Docker services"
	@echo "  $(GREEN)ollama-pull$(RESET)  - Pull Ollama model"
	@echo "  $(GREEN)logs$(RESET)         - Show Docker logs"
	@echo "  $(GREEN)dev$(RESET)          - Start development mode"
	@echo "  $(GREEN)help$(RESET)         - Show this help"
