FROM golang:1.22.6-bookworm as builder

WORKDIR /app
ENV CGO_ENABLED=0

COPY go.mod go.sum ./
RUN go mod download

COPY pkg/ ./pkg/
COPY main.go ./
RUN go build


FROM scratch as runner

WORKDIR /app
COPY --from=builder /app/api-notifications .
CMD [ "/app/api-notifications" ]
