FROM golang:1.25 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY main.go .
RUN CGO_ENABLED=0 go build -o extract .

FROM ubuntu:22.04

RUN apt-get update && apt-get install -y \
    wget \
    util-linux \
    && rm -rf /var/lib/apt/lists/*

# Install MakeMKV
RUN wget -q "https://www.makemkv.com/download/makemkv-oss-1.18.3.tar.gz" -O /tmp/makemkv-oss.tar.gz \
    && wget -q "https://www.makemkv.com/download/makemkv-bin-1.18.3.tar.gz" -O /tmp/makemkv-bin.tar.gz \
    && apt-get update && apt-get install -y \
        build-essential \
        pkg-config \
        libc6-dev \
        libssl-dev \
        libexpat1-dev \
        libavcodec-dev \
        libgl1-mesa-dev \
        qtbase5-dev \
        zlib1g-dev \
    && tar -xzf /tmp/makemkv-oss.tar.gz -C /tmp \
    && tar -xzf /tmp/makemkv-bin.tar.gz -C /tmp \
    && cd /tmp/makemkv-oss-1.18.3 && ./configure && make && make install \
    && cd /tmp/makemkv-bin-1.18.3 && mkdir -p tmp && touch tmp/eula_accepted && make && make install \
    && rm -rf /tmp/makemkv-* \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /app/extract /usr/local/bin/extract

ENTRYPOINT ["/usr/local/bin/extract"]
