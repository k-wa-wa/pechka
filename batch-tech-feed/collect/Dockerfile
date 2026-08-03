# テキスト収集・選定・一次情報検証・台本生成を行う軽量イメージ。
FROM golang:1.25-bookworm AS builder
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
ENTRYPOINT ["/usr/local/bin/tech-feed"]
CMD ["pipeline"]
