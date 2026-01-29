# --- Stage 1: Build ---
FROM golang:1.24 AS builder

WORKDIR /app

# Copy dependencies trước để cache
COPY go.mod go.sum ./
RUN go mod download

# Copy toàn bộ source vào container
COPY . .

# Build binary từ thư mục cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -o storage-service ./cmd/server

# --- Stage 2: Run (image tối ưu) ---
FROM alpine:3.20

WORKDIR /app

COPY --from=builder /app/storage-service .

EXPOSE 5003

CMD ["./storage-service"]
