APP     := app
BIN_DIR := bin
IMAGE   := grinex-adapter-service:latest

.PHONY: build test lint run docker-build generate

## build: compile a static binary into ./bin/app
build:
	CGO_ENABLED=0 go build -o $(BIN_DIR)/$(APP) ./cmd

## test: run unit tests
test:
	go test ./...

## lint: run golangci-lint
lint:
	golangci-lint run ./...

## run: run the service locally (needs postgres and jaeger: docker-compose up -d)
run:
	go run ./cmd

## docker-build: build the application image
docker-build:
	docker build -t $(IMAGE) .

## generate: regenerate gRPC code from proto
generate:
	buf generate