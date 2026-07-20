FROM bufbuild/buf:1.71 as buf

COPY common/proto /proto
WORKDIR /proto

RUN buf generate


FROM golang:1.26-alpine AS builder

ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

WORKDIR /app

COPY notification-service/ ./
COPY common/ ../common/
COPY --from=buf /proto/generated ../common/proto/generated

RUN --mount=type=cache,id=go-mod,target=/go/pkg/mod \
    --mount=type=cache,id=go-build,target=/root/.cache/go-build \
    go generate ./... && \
    go build -trimpath -ldflags="-s -w" -o /app/notification-service ./cmd


FROM alpine:3.22 AS upx

RUN --mount=type=cache,id=apk-cache,target=/var/cache/apk \
    apk add upx

COPY --from=builder /app/notification-service /app/notification-service

RUN upx --best --lzma /app/notification-service


FROM gcr.io/distroless/static-debian13:nonroot

WORKDIR /app

COPY --from=upx /app/notification-service .

EXPOSE 8085

ENTRYPOINT ["./notification-service"]
