TARGET = canary-router

.PHONY: get-tools
get-tools:
	GO111MODULE=off go get -u -v github.com/Arkweid/lefthook
	GO111MODULE=off go get -u -v golang.org/x/lint/golint
	GO111MODULE=off go get -u -v github.com/golangci/golangci-lint/cmd/golangci-lint

.PHONY: test
test:
	go test -v ./...

.PHONY: unit-test
unit-test:
	go test -v -short ./...

.PHONY: integration-test
integration-test:
	go test -v -run integration ./...

.PHONY: build
build:
	go build -v -o bin/$(TARGET) ./cmd/

.DEFAULT_GOAL := build

