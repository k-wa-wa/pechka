FROM golang:1.25 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ cmd/
RUN CGO_ENABLED=0 go build -o transcode ./cmd/transcode \
    && CGO_ENABLED=0 go build -o generate-master ./cmd/generate-master

FROM ubuntu:22.04

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get install -y \
    ffmpeg \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /app/transcode /usr/local/bin/transcode
COPY --from=builder /app/generate-master /usr/local/bin/generate-master

ENTRYPOINT ["/usr/local/bin/transcode"]
