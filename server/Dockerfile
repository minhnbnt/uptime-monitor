FROM golang:1.26-alpine AS proto-gen
RUN go install github.com/bufbuild/buf/cmd/buf@latest
COPY proto /tmp/proto
RUN cd /tmp/proto && buf generate
RUN mkdir -p /gen && cp -r /tmp/proto/gen/* /gen/

FROM golang:1.26-alpine AS builder

ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

WORKDIR /app/server

COPY server/go.mod server/go.sum ./
RUN --mount=type=cache,id=go-mod,target=/go/pkg/mod \
    go mod download

COPY server/ ./
COPY --from=proto-gen /gen /app/proto/gen

RUN --mount=type=cache,id=go-mod,target=/go/pkg/mod \
    --mount=type=cache,id=go-build,target=/root/.cache/go-build \
    go install github.com/ogen-go/ogen/cmd/ogen@latest && \
    go generate ./... && \
    go build -trimpath -ldflags="-s -w" -o /app/server/app ./cmd/main.go

FROM alpine:3.22 AS upx

RUN --mount=type=cache,id=apk-cache,target=/var/cache/apk \
    apk add upx

COPY --from=builder /app/server/app /app/app

RUN upx --best --lzma /app/app

FROM gcr.io/distroless/static-debian13:nonroot

WORKDIR /app

COPY --from=upx /app/app .

EXPOSE 8080

ENTRYPOINT ["./app"]
