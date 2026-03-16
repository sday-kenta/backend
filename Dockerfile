FROM golang:1.25-alpine3.21 AS modules

COPY go.mod go.sum /modules/
WORKDIR /modules
ENV GOPROXY=https://proxy.golang.org,direct
RUN go mod download -x

FROM golang:1.25-alpine3.21 AS builder

COPY --from=modules /go/pkg /go/pkg
COPY . /app
WORKDIR /app
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /bin/app ./cmd/app

FROM scratch
COPY --from=builder /app/config /config
COPY --from=builder /bin/app /app
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
CMD ["/app"]
