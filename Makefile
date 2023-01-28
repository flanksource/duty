test:
	go test   ./... -test.v

fmt:
	go fmt ./...

.PHONY: lint
lint:
	golangci-lint run

download-openapi-schemas:
	# Canary Checker
	curl https://raw.githubusercontent.com/flanksource/canary-checker/master/config/schemas/canary.schema.json -o schema/openapi/canary.schema.json
	curl https://raw.githubusercontent.com/flanksource/canary-checker/master/config/schemas/component.schema.json -o schema/openapi/component.schema.json
	curl https://raw.githubusercontent.com/flanksource/canary-checker/master/config/schemas/system.schema.json -o schema/openapi/system.schema.json
