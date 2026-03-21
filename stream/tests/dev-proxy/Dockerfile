FROM golang:alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/dev-proxy main.go

FROM alpine:latest
COPY --from=builder /bin/dev-proxy /bin/dev-proxy
ENTRYPOINT ["/bin/dev-proxy"]
