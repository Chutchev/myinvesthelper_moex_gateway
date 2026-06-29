.DEFAULT_GOAL := build

.PHONY: run test vet build fmt swagger

run:
	go run ./cmd/gateway

test:
	go test ./...

vet:
	go vet ./...

build:
	go build ./...

fmt:
	gofmt -w cmd internal

swagger:
	go tool swag init -g main.go -d cmd/gateway,internal/httpserver,internal/moex,internal/cbr -o docs --parseInternal
