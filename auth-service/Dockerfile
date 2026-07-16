FROM golang:1.26-alpine AS builder

ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

WORKDIR /app/auth-service

COPY . ./

RUN --mount=type=cache,id=go-mod,target=/go/pkg/mod \
    --mount=type=cache,id=go-build,target=/root/.cache/go-build \
    go generate ./...

RUN --mount=type=cache,id=go-mod,target=/go/pkg/mod \
    --mount=type=cache,id=go-build,target=/root/.cache/go-build \
    go build -trimpath -ldflags="-s -w" -o app ./cmd/main.go


FROM alpine:3.22 AS upx

RUN --mount=type=cache,id=apk-cache,target=/var/cache/apk \
    apk add upx

COPY --from=builder /app/auth-service/app /app/app

RUN upx --best --lzma /app/app


FROM gcr.io/distroless/static-debian13:nonroot

WORKDIR /app

COPY --from=upx /app/app .

EXPOSE 8080

ENTRYPOINT ["./app"]
