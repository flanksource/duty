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
	
	# create schemas for specs only
	cat tmp/canary-checker/config/schemas/canary.schema.json | jq '.["$$ref"] = "#/definitions/CanarySpec"' > schema/openapi/canary.specs.schema.json
	cat tmp/canary-checker/config/schemas/component.schema.json | jq '.["$$ref"] = "#/definitions/ComponentSpec"' > schema/openapi/component.specs.schema.json
	cat tmp/canary-checker/config/schemas/system.schema.json | jq '.["$$ref"] = "#/definitions/SystemTemplateSpec"' > schema/openapi/system.spec.schema.json
	
	# Config DB
	git clone --depth=1 git@github.com:flanksource/config-db.git tmp/config-db && cp tmp/config-db/config/schemas/* schema/openapi/
	
	# create schemas for specs only
	cat tmp/config-db/config/schemas/scrape_config.schema.json | jq '.["$$ref"] = "#/definitions/ScrapeConfigSpec"' > schema/openapi/scrape_config.spec.schema.json
	
	# Cleanup
	rm -rf tmp
