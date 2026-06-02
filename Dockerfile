FROM golang:1.26-alpine AS builder

WORKDIR /app

ENV CGO_ENABLED 0
ENV GOOS linux
ENV GOARCH amd64

COPY go.mod go.sum ./

RUN --mount=type=cache,id=go-mod,target=/go/pkg/mod \
    --mount=type=cache,id=go-build,target=/root/.cache/go-build \
    go mod download

COPY . .

RUN --mount=type=cache,id=go-mod,target=/go/pkg/mod \
    --mount=type=cache,id=go-build,target=/root/.cache/go-build \
    go generate ./cmd/main.go

RUN --mount=type=cache,id=go-mod,target=/go/pkg/mod \
    --mount=type=cache,id=go-build,target=/root/.cache/go-build \
    go build -ldflags="-s -w" -o app ./cmd/main.go

# -----------------------------------------------------

# FROM alpine:3.22 AS upx
#
# RUN --mount=type=cache,target=/var/cache/apk \
#     apk add --no-cache upx
#
# COPY --from=builder /app/app /app/app
#
# RUN upx --best --lzma /app/app

# -----------------------------------------------------

FROM gcr.io/distroless/static:latest

WORKDIR /app

COPY --from=builder /app/app .

EXPOSE 8080

CMD ["./app"]
