.PHONY: dev build build-dev generate test test-v test-cover test-cover-html format

dev:
	go tool air -c .air.toml

build:
	go build -ldflags="-s -w" -o bin/uptime-monitor ./cmd/main.go

build-dev:
	go build -o tmp/uptime-monitor ./cmd/main.go

generate:
	go generate ./...

test:
	go test -count=1 ./internal/...

test-v:
	go test -v -count=1 ./internal/...

test-cover:
	go test -cover -count=1 ./internal/...

test-cover-html:
	go test -coverprofile=/tmp/cover.out -count=1 ./internal/...
	go tool cover -html=/tmp/cover.out

format:
	golangci-lint run ./... --fix
