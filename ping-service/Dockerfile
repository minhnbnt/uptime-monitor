FROM bufbuild/buf:1.72.0 as buf

COPY common/proto /proto
WORKDIR /proto

# -----------------------------------------------------

RUN buf generate

FROM public.ecr.aws/docker/library/golang:1.26 AS builder

ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

WORKDIR /app/ping-service

COPY ping-service/ ./
COPY common/ ../common/
COPY --from=buf /proto/generated ../common/proto/generated

RUN --mount=type=cache,id=go-mod,target=/go/pkg/mod \
    --mount=type=cache,id=go-build,target=/root/.cache/go-build \
    go build -trimpath -ldflags="-s -w" -o app ./cmd/main.go

# -----------------------------------------------------

FROM public.ecr.aws/docker/library/alpine:3 AS upx

RUN --mount=type=cache,id=apk-cache,target=/var/cache/apk \
    apk add upx

COPY --from=builder /app/ping-service/app /app/app

RUN upx --best --lzma /app/app


FROM registry.access.redhat.com/hi/static:latest

WORKDIR /app

COPY --from=upx /app/app .

EXPOSE 8080

ENTRYPOINT ["./app"]
