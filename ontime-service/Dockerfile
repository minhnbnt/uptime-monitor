FROM bufbuild/buf:1.71 as buf

COPY common/proto /proto
WORKDIR /proto

RUN buf generate


FROM golang:1.26-alpine AS builder

ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

WORKDIR /app

COPY ontime-service/ ./
COPY common/ ../common/
COPY --from=buf /proto/generated ../common/proto/generated

RUN --mount=type=cache,id=go-mod,target=/go/pkg/mod \
    --mount=type=cache,id=go-build,target=/root/.cache/go-build \
    go generate ./... && \
    go build -trimpath -ldflags="-s -w" -o /app/ontime-service ./cmd


FROM alpine:3.22 AS upx

RUN --mount=type=cache,id=apk-cache,target=/var/cache/apk \
    apk add upx

COPY --from=builder /app/ontime-service /app/ontime-service

RUN upx --best --lzma /app/ontime-service


FROM gcr.io/distroless/static-debian13:nonroot

WORKDIR /app

COPY --from=upx /app/ontime-service .

EXPOSE 8084

ENTRYPOINT ["./ontime-service"]
