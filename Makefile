.PHONY: build test lint run

build:
	go build -o bin/urlwatch ./cmd/urlwatch

test:
	go test -race -count=1 ./...

lint:
	go vet ./...
	@test -z "$$(gofmt -l .)" || (echo "gofmt needed:"; gofmt -l .; exit 1)

run:
	go run ./cmd/urlwatch $(ARGS)
