.PHONY: build test test-race test-cover cover cover-html vet lint fix docker docker-up docker-dev clean

build:
	go build -o laightdb ./cmd/laightdb

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
	rm -f laightdb coverage.out coverage.html
