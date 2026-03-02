.PHONY: build test lint run docker-build tidy

build:
	CGO_ENABLED=0 go build -o bin/mock-server ./cmd/server

test:
	go test -race -count=1 -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out | tail -1

lint:
	golangci-lint run ./...

run:
	go run ./cmd/server --config=config/default.yaml

tidy:
	go mod tidy

docker-build:
	docker build -f deploy/Dockerfile -t google-billing-mock:latest .

clean:
	rm -f bin/mock-server coverage.out
