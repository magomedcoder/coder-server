.PHONY: build run test

build:
	mkdir -p ./build
	go build -o ./build/coder-server ./cmd/coder-server

run:
	go run ./cmd/coder-server

test:
	go test ./... -race -count=1
