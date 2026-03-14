# --- Stage 1: Build ---
FROM golang:1.26 AS builder

WORKDIR /app

# Download ONNX Runtime shared library (Linux x86-64)
ARG ONNX_VERSION=1.20.1
RUN apt-get update -qq && apt-get install -y -q wget \
    && wget -q "https://github.com/microsoft/onnxruntime/releases/download/v${ONNX_VERSION}/onnxruntime-linux-x64-${ONNX_VERSION}.tgz" \
    && tar -xzf onnxruntime-linux-x64-${ONNX_VERSION}.tgz \
    && mv onnxruntime-linux-x64-${ONNX_VERSION}/lib/libonnxruntime.so.${ONNX_VERSION} /app/libonnxruntime.so \
    && rm -rf onnxruntime-linux-x64-${ONNX_VERSION}*

# Copy dependencies first to leverage layer cache
COPY go.mod go.sum ./
RUN go mod download

# Copy full source
COPY . .

# Build binary with CGO enabled (required by onnxruntime_go)
RUN CGO_ENABLED=1 GOOS=linux go build -o storage-service ./cmd/server

# --- Stage 2: Run ---
FROM debian:bookworm-slim

WORKDIR /app

# Copy ONNX Runtime shared library
COPY --from=builder /app/libonnxruntime.so /app/libonnxruntime.so

# Copy binary and migration files
COPY --from=builder /app/storage-service .
COPY --from=builder /app/migration ./migration

# Copy ML model
COPY models/ ./models/

EXPOSE 5003

ENV ONNX_LIB_PATH=/app/libonnxruntime.so
ENV MODEL_PATH=/app/models/readiness_model.onnx

CMD ["./storage-service"]
