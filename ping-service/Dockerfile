FROM golang:1.26-alpine AS proto-gen
RUN go install github.com/bufbuild/buf/cmd/buf@latest
COPY proto /tmp/proto
RUN cd /tmp/proto && buf generate
RUN mkdir -p /gen && cp -r /tmp/proto/gen/* /gen/

FROM golang:1.26-alpine AS builder

ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

WORKDIR /app/ping-service

COPY ping-service/go.mod ping-service/go.sum ./
RUN --mount=type=cache,id=go-mod-ping,target=/go/pkg/mod \
    go mod download

COPY ping-service/ ./
COPY --from=proto-gen /gen /app/proto/gen

RUN --mount=type=cache,id=go-mod-ping,target=/go/pkg/mod \
    --mount=type=cache,id=go-build-ping,target=/root/.cache/go-build \
    go build -trimpath -ldflags="-s -w" -o /app/ping-service/app ./cmd/main.go

FROM alpine:3.21

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /app/ping-service/app .
COPY ping-service/config.yml /config.yml

EXPOSE 8083

ENTRYPOINT ["/app/app"]
CMD ["--config", "/config.yml"]
