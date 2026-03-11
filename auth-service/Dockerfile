# --- Stage 1: Build ---
FROM golang:1.24 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o auth-service ./cmd/server

# --- Stage 2: Run (image tối ưu) ---
FROM alpine:3.20

WORKDIR /app

# Cần CA cert cho kết nối HTTPS
RUN apk --no-cache add ca-certificates

COPY --from=builder /app/auth-service .

EXPOSE 5000

CMD ["./auth-service"]
