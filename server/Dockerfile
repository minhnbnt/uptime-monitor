FROM bufbuild/buf:1.71 as buf

COPY common/proto /proto
WORKDIR /proto

RUN buf generate


FROM golang:1.26-alpine AS builder

ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

WORKDIR /app/server

COPY server/ ./
COPY common/ ../common/
COPY --from=buf /proto/generated ../common/proto/generated

RUN --mount=type=cache,id=go-mod,target=/go/pkg/mod \
    --mount=type=cache,id=go-build,target=/root/.cache/go-build \
    go generate ./... && \
    go build -trimpath -ldflags="-s -w" -o /app/server/app ./cmd/main.go


FROM gcr.io/distroless/static-debian13:nonroot

WORKDIR /app

COPY --from=builder /app/server/app .

EXPOSE 8080

ENTRYPOINT ["./app"]
