# Build frontend
FROM node:22-alpine AS frontend
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# Build Go binary
FROM golang:1.26-alpine AS builder
RUN apk add --no-cache git
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /app/frontend/dist ./internal/server/frontend/dist
RUN go build -ldflags "-X github.com/SEObserver/crawlobserver/internal/updater.Version=railway" -o crawlobserver ./cmd/crawlobserver

# Runtime
FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/crawlobserver .
COPY --from=builder /app/migrations ./migrations
COPY --from=builder /app/config.railway.yaml ./config.yaml
EXPOSE 8899
CMD ["./crawlobserver", "serve"]
