FROM bufbuild/buf:1.71 as buf

COPY proto /proto
WORKDIR /proto

# -----------------------------------------------------

RUN buf generate

FROM golang:1.26-alpine AS builder

ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

WORKDIR /app/ping-service

COPY ping-service/ ./
COPY --from=buf /proto/gen ../proto/gen

RUN --mount=type=cache,id=go-mod,target=/go/pkg/mod \
    --mount=type=cache,id=go-build,target=/root/.cache/go-build \
    go build -trimpath -ldflags="-s -w" -o app ./cmd/main.go

# -----------------------------------------------------

FROM gcr.io/distroless/static-debian13:nonroot

WORKDIR /app

COPY --from=builder /app/ping-service/app .

EXPOSE 8080

ENTRYPOINT ["./app"]
