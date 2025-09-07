.PHONY: build clean run test deps install daemon stop logs help

# Default target
help: ## Show help
	@echo "GitHub-Todoist Synchronization"
	@echo "Available commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

# Build
build: ## Build the application
	@echo "Building the application..."
	go build -o bin/github-todoist-sync cmd/sync/main.go
	@echo "Application built: bin/github-todoist-sync"

# Dependencies
deps: ## Download dependencies
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy

# Installation (only if we want to install globally)
install: build ## Install the application to $GOPATH/bin
	@echo "Installing the application..."
	go install cmd/sync/main.go

# Run - one-time synchronization
run: build ## Run one-time synchronization
	@echo "Starting one-time synchronization..."
	./bin/github-todoist-sync -mode=once -verbose

# Run - GitHub -> Todoist only
github-sync: build ## Synchronize only from GitHub to Todoist
	@echo "Synchronizing GitHub → Todoist..."
	./bin/github-todoist-sync -mode=github-only -verbose

# Run - Todoist -> GitHub only
todoist-sync: build ## Synchronize only from Todoist to GitHub
	@echo "Synchronizing Todoist → GitHub..."
	./bin/github-todoist-sync -mode=todoist-only -verbose

# Run as a daemon
daemon: build ## Run the application as a daemon (in the background)
	@echo "Starting daemon..."
	nohup ./bin/github-todoist-sync -mode=daemon -verbose > logs/sync.log 2>&1 & echo $$! > .daemon.pid
	@echo "Daemon started with PID: $$(cat .daemon.pid)"
	@echo "Logs: tail -f logs/sync.log"

# Stop daemon
stop: ## Stop daemon
	@if [ -f .daemon.pid ]; then \
		PID=$$(cat .daemon.pid); \
		if ps -p $$PID > /dev/null 2>&1; then \
			echo "Stopping daemon (PID: $$PID)..."; \
			kill $$PID; \
			sleep 2; \
			if ps -p $$PID > /dev/null 2>&1; then \
				echo "Daemon not responding, forcing termination..."; \
				kill -9 $$PID; \
			fi; \
		else \
			echo "Daemon is not running"; \
		fi; \
		rm -f .daemon.pid; \
	else \
		echo "Daemon is not running (.daemon.pid does not exist)"; \
	fi

# Show logs
logs: ## Show daemon logs
	@if [ -f logs/sync.log ]; then \
		tail -f logs/sync.log; \
	else \
		echo "Log file does not exist. Start the daemon first: make daemon"; \
	fi

# Tests
test: ## Run tests
	@echo "Running tests..."
	go test -v ./...

# Clean
clean: ## Clean build artifacts
	@echo "Cleaning..."
	rm -f bin/github-todoist-sync
	rm -f .daemon.pid
	rm -f logs/sync.log

# Environment setup
setup: ## Prepare environment (create directories, copy .env)
	@echo "Preparing environment..."
	mkdir -p bin logs
	@if [ ! -f .env ]; then \
		cp .env.example .env; \
		echo "Created .env file from template. Please fill in the API tokens."; \
	else \
		echo ".env file already exists"; \
	fi
	@echo "Environment ready!"

# Status
status: ## Show daemon status
	@if [ -f .daemon.pid ]; then \
		PID=$$(cat .daemon.pid); \
		if ps -p $$PID > /dev/null 2>&1; then \
			echo "Daemon is running with PID: $$PID"; \
			echo "Start time: $$(ps -o lstart= -p $$PID)"; \
		else \
			echo "Daemon is not running (PID file exists, but the process is not running)"; \
			rm -f .daemon.pid; \
		fi; \
	else \
		echo "Daemon is not running"; \
	fi

# Quick setup for development
dev: setup deps build ## Prepare everything for development