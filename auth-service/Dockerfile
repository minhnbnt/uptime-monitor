FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /auth-service ./cmd

FROM alpine:3.21
RUN apk --no-cache add ca-certificates
COPY --from=builder /auth-service /auth-service
COPY config.yaml /config.yaml
EXPOSE 8081
ENTRYPOINT ["/auth-service"]
CMD ["--config", "/config.yaml"]
