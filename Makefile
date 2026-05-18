.PHONY: dev build generate

dev:
	go run github.com/air-verse/air -c .air.toml

build:
	go build -o bin/uptime-monitor ./cmd/main.go

build-dev:
	go build -o tmp/uptime-monitor ./cmd/main.go

generate:
	go generate ./...
