.PHONY: build test tidy
build:
	go build -o bin/depscount ./cmd/depscount/

test:
	go test ./...

tidy:
	go mod tidy
