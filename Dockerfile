# Stage 1: Build frontend
FROM node:22-alpine AS frontend
WORKDIR /app/frontend
RUN corepack enable && corepack prepare pnpm@latest --activate
COPY frontend/package.json frontend/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile
COPY frontend/ ./
RUN pnpm run build

# Stage 2: Build Go binary
FROM golang:1.25-alpine AS backend
ARG VERSION=dev
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /app/internal/server/dist/ ./internal/server/dist/
RUN go build -ldflags "-X main.version=${VERSION}" -o fichemos ./cmd/server

# Stage 3: Final image
FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata exiftool
WORKDIR /app

COPY --from=backend /app/fichemos .
COPY --from=backend /app/db/migrations/ ./db/migrations/

RUN mkdir -p /data

EXPOSE 8080

CMD ["./fichemos"]
