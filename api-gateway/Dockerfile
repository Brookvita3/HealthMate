# ===== BUILD STAGE =====
FROM golang:1.24 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o api-gateway ./cmd/server

FROM alpine:3.20

WORKDIR /app

COPY --from=builder /app/api-gateway .

EXPOSE 5002

CMD ["./api-gateway"]