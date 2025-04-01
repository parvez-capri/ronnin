.PHONY: build run test clean

build:
	go build -o bin/api ./cmd/api

run:
	go run ./cmd/api

test:
	go test ./... -v

clean:
	rm -rf bin/