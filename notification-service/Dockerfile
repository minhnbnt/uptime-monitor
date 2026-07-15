FROM golang:1.26-alpine AS builder

ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

WORKDIR /app

COPY notification-service/ ./
COPY common/ ../common/

RUN --mount=type=cache,id=go-mod,target=/go/pkg/mod \
    --mount=type=cache,id=go-build,target=/root/.cache/go-build \
    go generate ./... && \
    go build -trimpath -ldflags="-s -w" -o /app/notification-service ./cmd


FROM gcr.io/distroless/static-debian13:nonroot

WORKDIR /app

COPY --from=builder /app/notification-service .

EXPOSE 8085

ENTRYPOINT ["./notification-service"]
