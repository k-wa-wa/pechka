FROM golang:1.22.6-bookworm as builder
ARG TARGET

WORKDIR /app
ENV CGO_ENABLED=0

COPY go.mod go.sum ./
RUN go mod download

COPY cmd/ ./cmd/
COPY pkg/ ./pkg/
COPY internal/ ./internal/
RUN go build -o bin ./cmd/${TARGET}

FROM scratch as runner

WORKDIR /app
COPY --from=builder /app/bin .

CMD [ "/app/bin" ]
