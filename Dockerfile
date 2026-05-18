FROM golang:1.26-alpine AS builder

WORKDIR /app

ENV CGO_ENABLED 0
ENV GOOS linux
ENV GOARCH amd64

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

COPY . .

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -ldflags="-s -w" -o app ./cmd/main.go

FROM gcr.io/distroless/static:latest

WORKDIR /app

COPY --from=builder /app/app .

EXPOSE 8080

CMD ["./app"]
