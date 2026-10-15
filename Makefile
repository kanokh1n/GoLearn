.PHONY: run build test test-short lint

run:
	go run ./cmd/golearn

build:
	go build -o bin/golearn ./cmd/golearn

test:
	go test ./...

# Skips tests that compile Go code (lesson content checks).
test-short:
	go test -short ./...

lint:
	go vet ./...
	gofmt -l .
