FROM golang:1.24.2-bookworm AS builder

WORKDIR /app
ENV CGO_ENABLED=0

COPY go.mod go.sum ./
RUN go mod download

COPY cmd/ ./cmd/
COPY pkg/ ./pkg/
COPY internal/ ./internal/

ARG TARGET
RUN go build -o bin ./cmd/${TARGET}


FROM scratch AS runner

WORKDIR /app
COPY --from=builder /app/bin .

CMD [ "/app/bin" ]
