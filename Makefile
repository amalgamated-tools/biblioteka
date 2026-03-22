.PHONY: all build frontend backend clean dev redis-check screenshots kill-dev swagger swagger-fmt

# Tooling commands
SWAG_CMD = go run github.com/swaggo/swag/v2/cmd/swag@v2.0.0-rc5
GOLANGCI_LINT_CMD = go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@1c222b488bbc2c0ae2cad8423a24b8452f2fc3a9

# Build everything: frontend then Go binary
all: build

# Install frontend dependencies
frontend/node_modules: frontend/package.json
	cd frontend && pnpm install

# Install root-level tooling dependencies
node_modules: package.json
	pnpm install

# Build the frontend into internal/server/dist/
frontend: frontend/node_modules
	cd frontend && pnpm run build && touch ../internal/server/dist/.gitkeep

# Build the Go binary (requires frontend build first)
backend: frontend
	go build -o biblioteka ./cmd/server

# Build everything
build: backend

# Run the Go server (production mode)
run: build
	./biblioteka

# Run frontend and backend dev servers via goreman
dev: redis-check frontend/node_modules kill-dev
	goreman -f Procfile.dev start

kill-dev:
	@echo "Stopping any existing servers on ports 5173 and 8080..."
	@for port in 5173 8080; do \
		pids=$$(lsof -ti :$$port); \
		if [ -n "$$pids" ]; then \
			echo "Sending SIGTERM to processes on port $$port: $$pids"; \
			kill $$pids 2>/dev/null || true; \
		fi; \
	done
	@sleep 2
	@for port in 5173 8080; do \
		pids=$$(lsof -ti :$$port); \
		if [ -n "$$pids" ]; then \
			echo "Sending SIGKILL to remaining processes on port $$port: $$pids"; \
			kill -9 $$pids 2>/dev/null || true; \
		fi; \
	done
	@echo "Dev ports 5173 and 8080 are now free."

# Capture application screenshots via Playwright
screenshots: clean node_modules frontend/node_modules
	@mkdir -p screenshots
	# call kill-dev first to ensure no existing servers are running
	$(MAKE) kill-dev
	@echo "Starting dev server in background..."	
	@goreman -f Procfile.screen start & DEV_PID=$$!; \
	echo "Waiting for frontend (localhost:5173) and backend (localhost:8080)..."; \
	for i in $$(seq 1 60); do \
		if curl -s -o /dev/null http://localhost:5173 && curl -s -o /dev/null http://localhost:8080/health; then \
			echo "Servers ready."; \
			break; \
		fi; \
		if [ $$i -eq 60 ]; then \
			echo "Timeout waiting for servers"; \
			kill $$DEV_PID 2>/dev/null; \
			exit 1; \
		fi; \
		sleep 1; \
	done; \
	node script/take-screenshots.mjs; \
	RESULT=$$?; \
	kill $$DEV_PID 2>/dev/null; \
	exit $$RESULT

redis-check:
	@set -e; \
	if command -v redis-cli > /dev/null 2>&1; then \
		if redis-cli ping > /dev/null 2>&1; then \
			echo "Redis is reachable via redis-cli."; \
			exit 0; \
		fi; \
	fi; \
	if command -v docker > /dev/null 2>&1; then \
		if docker ps --filter "name=redis" --filter "status=running" -q | grep -q .; then \
			if docker compose ps --status=running redis 2>/dev/null | grep -q redis; then \
				echo "Redis Docker container is running via docker-compose."; \
				exit 0; \
			elif docker ps --filter "name=redis" --filter "status=running" --filter "publish=6379" -q | grep -q .; then \
				echo "A non-compose Redis container is running with port 6379 mapped."; \
				exit 0; \
			else \
				echo "Warning: A container with 'redis' in its name is running, but it is not managed by this project's docker-compose.yml and does not have port 6379 mapped."; \
			fi; \
		fi; \
	fi; \
	if command -v nc > /dev/null 2>&1; then \
		if nc -z localhost 6379 > /dev/null 2>&1; then \
			echo "Something is listening on port 6379. Assuming Redis is running."; \
			exit 0; \
		fi; \
	fi; \
	echo "Error: Redis is not running. Start it with 'redis-server', 'brew services start redis', or 'docker compose up -d redis'."; \
	exit 1

# Clean build artifacts
clean:
	rm -rf internal/server/dist tmp biblioteka db/biblioteka.db*
	mkdir -p internal/server/dist
	touch internal/server/dist/.gitkeep

# Generate Swagger/OpenAPI documentation
swagger:
	$(SWAG_CMD) init -g cmd/server/main.go -o docs --parseDependency --parseInternal

# Format swagger annotations
swagger-fmt:
	$(SWAG_CMD) fmt

lint:
	$(GOLANGCI_LINT_CMD) run ./... --max-issues-per-linter 0 --max-same-issues 0

fmt:
	go fmt ./...

hardfmt:
	go tool gofumpt -w -l .

test:
	go test -v ./...

testsum:
	gotestsum -- -v ./...

modernize:
	go run golang.org/x/tools/go/analysis/passes/modernize/cmd/modernize@latest -fix  ./...