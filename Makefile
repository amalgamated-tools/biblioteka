.PHONY: all build frontend backend clean dev redis-check screenshots

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
dev: redis-check frontend/node_modules
	goreman -f Procfile.dev start

# Capture application screenshots via Playwright
screenshots: node_modules
	@mkdir -p screenshots
	@echo "Killing any existing servers on ports 5173 and 8080..."
	@-lsof -ti :5173 | xargs kill -9 2>/dev/null || true
	@-lsof -ti :8080 | xargs kill -9 2>/dev/null || true
	@echo "Starting dev server in background..."	
	@goreman -f Procfile.dev start & DEV_PID=$$!; \
# 	echo "Waiting for frontend (localhost:5173) and backend (localhost:8080)..."; \
# 	for i in $$(seq 1 60); do \
# 		if curl -s -o /dev/null http://localhost:5173 && curl -s -o /dev/null http://localhost:8080/health; then \
# 			echo "Servers ready."; \
# 			break; \
# 		fi; \
# 		if [ $$i -eq 60 ]; then \
# 			echo "Timeout waiting for servers"; \
# 			kill $$DEV_PID 2>/dev/null; \
# 			exit 1; \
# 		fi; \
# 		sleep 1; \
# 	node script/take-screenshots.mjs; \
# 	RESULT=$$?; \
# 	kill $$DEV_PID 2>/dev/null; \
# 	exit $$RESULT

redis-check:
	@set -e; \
	if command -v redis-cli > /dev/null 2>&1; then \
		redis-cli ping > /dev/null 2>&1 || (echo "Error: Redis is not running. Start it with 'redis-server' or 'brew services start redis'." && exit 1); \
	elif command -v docker > /dev/null 2>&1 && docker compose ps --status=running redis > /dev/null 2>&1; then \
		docker compose exec -T redis redis-cli ping > /dev/null 2>&1 || (echo "Error: Redis container is not reachable. Start it with 'docker compose up -d redis'." && exit 1); \
	else \
		echo "Error: redis-cli is not installed and Docker Redis check is unavailable. Install redis-cli or start Redis with 'docker compose up -d redis'."; \
		exit 1; \
	fi

# Clean build artifacts
clean:
	rm -rf internal/server/dist tmp biblioteka db/biblioteka.db*
	mkdir -p internal/server/dist
	touch internal/server/dist/.gitkeep
