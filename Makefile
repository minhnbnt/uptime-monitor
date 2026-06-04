.PHONY: dev build build-dev generate test test-v test-cover test-cover-html

dev:
	go run github.com/air-verse/air -c .air.toml

build:
	go build -ldflags="-s -w" -o bin/uptime-monitor ./cmd/main.go

build-dev:
	go build -o tmp/uptime-monitor ./cmd/main.go

generate:
	go generate ./cmd/main.go

test:
	go test -count=1 ./internal/...

test-v:
	go test -v -count=1 ./internal/...

test-cover:
	go test -cover -count=1 ./internal/...

test-cover-html:
	go test -coverprofile=/tmp/cover.out -count=1 ./internal/...
	go tool cover -html=/tmp/cover.out
