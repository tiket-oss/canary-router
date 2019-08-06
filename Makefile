TARGET = canary-router

.PHONY: get-dev-deps
get-tools:
	GO111MODULE=off go get -u -v github.com/Arkweid/lefthook
	GO111MODULE=off go get -u -v golang.org/x/lint/golint

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

