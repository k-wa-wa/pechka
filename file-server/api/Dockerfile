FROM golang:1.22.6-bookworm as builder

WORKDIR /app
ENV CGO_ENABLED=0

COPY go.mod go.sum ./
RUN go mod download

COPY internal ./internal
COPY main.go ./
RUN go build
CMD ["go", "run", "main.go"]

#FROM scratch as runner
#
#WORKDIR /app
#COPY --from=builder /app/file-server .
#CMD [ "/app/file-server" ]
