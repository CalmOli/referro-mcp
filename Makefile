.PHONY: build test run clean

build:
	go build -o referro-mcp ./cmd/referro-mcp

test:
	go test ./...

run:
	go run ./cmd/referro-mcp

clean:
	rm -f referro-mcp
