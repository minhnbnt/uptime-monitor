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


FROM gcr.io/distroless/static-debian13:nonroot

WORKDIR /app

COPY --from=builder /app/auth-service/app .

EXPOSE 8080

ENTRYPOINT ["./app"]
