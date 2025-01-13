FROM golang:1.23 as builder

WORKDIR /usr/src/app

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY main.go ./
COPY internal/ ./internal
RUN CGO_ENABLED=0 go build -o /usr/local/bin/app


FROM scratch

USER 0
COPY --from=builder /usr/local/bin/app /usr/local/bin/app

CMD [ "/usr/local/bin/app" ]
