test:
	go test   ./... -test.v

fmt:
	go fmt ./...

.PHONY: lint
lint:
	golangci-lint run

download-openapi-schemas:
	mkdir -p tmp
	# Canary Checker
	git clone --depth=1 git@github.com:flanksource/canary-checker.git tmp/canary-checker && cp tmp/canary-checker/config/schemas/* schema/openapi/
	# Config DB
	git clone --depth=1 git@github.com:flanksource/config-db.git tmp/config-db && cp tmp/config-db/config/schemas/* schema/openapi/
	# Cleanup
	rm -rf tmp
