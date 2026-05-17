FROM golang:1.26 AS builder

WORKDIR /app

ARG ONNX_VERSION=1.21.0

RUN apt-get update -qq && apt-get install -y --no-install-recommends \
    wget ca-certificates \
 && rm -rf /var/lib/apt/lists/*

RUN wget -q "https://github.com/microsoft/onnxruntime/releases/download/v${ONNX_VERSION}/onnxruntime-linux-x64-${ONNX_VERSION}.tgz" \
 && tar -xzf onnxruntime-linux-x64-${ONNX_VERSION}.tgz \
 && mv onnxruntime-linux-x64-${ONNX_VERSION}/lib/libonnxruntime.so.${ONNX_VERSION} /app/libonnxruntime.so \
 && rm -rf onnxruntime-linux-x64-${ONNX_VERSION}*

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o storage-service ./cmd/server


FROM debian:bookworm-slim

WORKDIR /app

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    libstdc++6 \
    libgcc-s1 \
 && rm -rf /var/lib/apt/lists/*

COPY --from=builder /app/storage-service ./
COPY --from=builder /app/migration ./migration
COPY --from=builder /app/models ./models
COPY --from=builder /app/libonnxruntime.so* ./

ENV LD_LIBRARY_PATH=/app
ENV ONNX_LIB_PATH=/app/libonnxruntime.so
ENV MODEL_PATH=/app/models/readiness_model.onnx
ENV STRESS_MODEL_PATH=/app/models/wesad_stress_hgbc.onnx
ENV STRESS_NORM_STATS_PATH=/app/models/wesad_norm_stats.json

EXPOSE 5003

CMD ["./storage-service"]
