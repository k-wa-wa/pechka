# 1. Build Go binary
FROM golang:1.25-bookworm AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /produce main.go

# 2. Runtime image with Node.js, Remotion Chrome, ffmpeg
FROM node:22-bookworm-slim

# fonts-noto-cjk と Remotion (Chrome Headless) の動的依存ライブラリを追加
RUN apt-get update && apt-get install -y --no-install-recommends \
    ffmpeg \
    fonts-noto-cjk \
    libnspr4 \
    libnss3 \
    libatk1.0-0 \
    libatk-bridge2.0-0 \
    libcups2 \
    libdrm2 \
    libdbus-1-3 \
    libxkbcommon0 \
    libxcomposite1 \
    libxdamage1 \
    libxfixes3 \
    libxrandr2 \
    libgbm1 \
    libasound2 \
    libpango-1.0-0 \
    libcairo2 \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY remotion/package.json remotion/package-lock.json ./remotion/
RUN cd remotion && npm ci && npx remotion browser ensure

COPY remotion/ ./remotion/
COPY examples/ ./examples/
COPY --from=builder /produce ./produce

ENTRYPOINT ["./produce"]
