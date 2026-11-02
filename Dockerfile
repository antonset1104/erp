# ==========================================
# Multi-stage Dockerfile for GoERP
# ==========================================

# Stage 1: Build binary with CGO (for SQLite / PostgreSQL)
FROM golang:1.22-alpine AS builder

WORKDIR /build

# Install build dependencies for CGO (gcc, musl-dev)
RUN apk add --no-cache gcc musl-dev git

# Download Go dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build statically compiled ERP server binary
ENV CGO_ENABLED=1
ENV GOOS=linux
RUN go build -ldflags="-w -s" -o /build/erp-server ./cmd/server/main.go

# Stage 2: Minimal runtime image
FROM alpine:3.19

WORKDIR /app

# Install ca-certificates and tzdata for accurate business timestamps
RUN apk add --no-cache ca-certificates tzdata

# Copy compiled binary from builder
COPY --from=builder /build/erp-server /app/erp-server

# Copy static web assets
COPY --from=builder /build/web /app/web

# Expose HTTP port
EXPOSE 8080

# Environment defaults
ENV PORT=8080
ENV DB_TYPE=sqlite
ENV DB_PATH=/app/erp.db
ENV JWT_SECRET=production-secret-replace-me

# Run ERP server
CMD ["/app/erp-server"]
