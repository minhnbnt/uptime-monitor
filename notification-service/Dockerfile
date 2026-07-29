FROM bufbuild/buf:1.71 as buf

COPY common/proto /proto
WORKDIR /proto

RUN buf generate


FROM public.ecr.aws/docker/library/golang:1.26 AS builder

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


FROM public.ecr.aws/docker/library/alpine:3 AS upx

RUN --mount=type=cache,id=apk-cache,target=/var/cache/apk \
    apk add upx

COPY --from=builder /app/notification-service /app/notification-service

RUN upx --best --lzma /app/notification-service


FROM registry.access.redhat.com/hi/static:latest

WORKDIR /app

COPY --from=upx /app/notification-service .

EXPOSE 8085

ENTRYPOINT ["./notification-service"]
