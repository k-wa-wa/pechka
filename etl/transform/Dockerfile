FROM golang:1.22 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY main.go .
RUN CGO_ENABLED=0 go build -o transform .

FROM ubuntu:22.04

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get install -y \
    ffmpeg \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /app/transform /usr/local/bin/transform

ENTRYPOINT ["/usr/local/bin/transform"]
