.DEFAULT_GOAL := build

.PHONY: run test vet build fmt

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
