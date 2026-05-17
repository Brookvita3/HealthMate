# --- Stage 1: Build ---
FROM golang:1.24 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o realtime-service ./cmd/server

# --- Stage 2: Run ---
FROM alpine:3.20

WORKDIR /app

RUN apk --no-cache add ca-certificates

COPY --from=builder /app/realtime-service .

EXPOSE 5001

CMD ["./realtime-service"]
