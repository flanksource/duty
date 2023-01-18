test:
	go test   ./... -test.v

fmt:
	go fmt ./...

.PHONY: lint
lint:
	golangci-lint run
