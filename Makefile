DEPS = $(shell go list -f '{{range .TestImports}}{{.}} {{end}}' ./...)
PACKAGES = $(shell go list ./...)

all: deps format install

cov:
	gocov test ./... | gocov-html > /tmp/coverage.html
	open /tmp/coverage.html

deps:
	@echo "--> Installing build dependencies"
	@go get -d -v ./...
	@echo $(DEPS) | xargs -n1 go get -d

install: deps
	@go install

test: deps
	go list ./... | xargs -n1 go test

format: deps
	@echo "--> Running go fmt"
	@go fmt $(PACKAGES)

.PHONY: all cov deps install test
