.PHONY: dev build build-dev generate test test-v test-cover test-cover-html test-integration format

dev:
	go tool air -c .air.toml

build:
	go build -ldflags="-s -w" -o bin/uptime-monitor ./cmd/main.go

build-dev:
	go build -o tmp/uptime-monitor ./cmd/main.go

generate:
	go generate ./...

test:
	go test -short -count=1 ./internal/...

test-v:
	go test -short -v -count=1 ./internal/...

test-cover:
	go test -short -cover -count=1 ./internal/...

test-cover-html:
	go test -short -coverprofile=/tmp/cover.out -count=1 ./internal/...
	go tool cover -html=/tmp/cover.out

test-integration:
	go test -count=1 -v ./internal/...

format:
	golangci-lint run ./... --fix
