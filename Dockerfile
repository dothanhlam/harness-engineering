# ── Stage 1: Build the Go binary ──────────────────────────────────────────────
FROM golang:1.26-alpine AS builder

WORKDIR /src

# Cache dependencies first
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o /harness_bin .

# ── Stage 2: Minimal runtime ─────────────────────────────────────────────────
FROM alpine:3.21

RUN apk add --no-cache curl jq bash

WORKDIR /app

# Copy the compiled binary
COPY --from=builder /harness_bin ./harness_bin

# Copy runtime config and agent prompts
COPY harness_config.json ./
COPY .agents/ ./.agents/

# Copy entrypoint script
COPY scripts/docker-entrypoint.sh ./entrypoint.sh
RUN chmod +x ./entrypoint.sh

# Create directories that will be volume-mounted
RUN mkdir -p /app/workspace /app/memory

ENTRYPOINT ["./entrypoint.sh"]
