.PHONY: build build-dev-mcp build-stress test test-race test-cover cover cover-html vet lint fix docker docker-up docker-dev clean

build:
	go build -o laightdb ./cmd/laightdb

# Development-only MCP (stdio): inspect data dir, WAL, auth summary — do not expose to production
build-dev-mcp:
	go build -o laightdb-dev-mcp ./cmd/laightdb-dev-mcp

# HTTP load against a running server (see cmd/laightdb-stress)
build-stress:
	go build -o laightdb-stress ./cmd/laightdb-stress

test:
	go test ./...

test-race:
	go test -race ./...

test-cover:
	go test -cover ./...

cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

cover-html:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

vet:
	go vet ./...

lint:
	go tool golangci-lint run

fix:
	go fix ./...

docker:
	docker compose build

docker-up:
	docker compose up -d

docker-dev:
	docker compose --profile dev up laightdb-dev

clean:
	rm -f laightdb laightdb-dev-mcp laightdb-stress coverage.out coverage.html
