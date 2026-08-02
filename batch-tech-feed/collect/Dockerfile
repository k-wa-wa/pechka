# 収集だけを担う小さなイメージ。ネットワーク待ちが主な処理なので、
# Python + Node + Chrome を積んだ produce 側のイメージを持ち出す必要がない。
# 外部ストレージも DB も触らないため、認証情報も要らない。
FROM golang:1.26-bookworm AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY main.go ./
COPY cmd/ ./cmd/
COPY shared/ ./shared/
RUN CGO_ENABLED=0 go build -o tech-feed .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /app/tech-feed /usr/local/bin/tech-feed
COPY sources.json /etc/tech-feed/sources.json
ENTRYPOINT ["/usr/local/bin/tech-feed", "collect"]
