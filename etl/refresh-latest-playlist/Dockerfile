FROM golang:1.22 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY main.go .
RUN CGO_ENABLED=0 GOOS=linux go build -o /refresh-latest-playlist .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /refresh-latest-playlist /usr/local/bin/refresh-latest-playlist
ENTRYPOINT ["/usr/local/bin/refresh-latest-playlist"]
