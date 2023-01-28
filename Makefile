test:
	go test   ./... -test.v

fmt:
	go fmt ./...

.PHONY: lint
lint:
	golangci-lint run

download-openapi-schemas:
	# Canary Checker
	mkdir -p tmp
	git clone --depth=1 git@github.com:flanksource/canary-checker.git tmp/canary-checker && cp tmp/canary-checker/config/schemas/* schema/openapi/
	rm -rf tmp
