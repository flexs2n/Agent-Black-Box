.PHONY: build test lint clean run

build:
	go build -o /bin/server ./cmd/server

test:
	go test ./...

lint:
	golangci-lint run

clean:
	go clean ./...
	rm -f /bin/server

run:
	go run ./cmd/server