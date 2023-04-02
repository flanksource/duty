test:
	go test   ./... -test.v

fmt:
	go fmt ./...

.PHONY: lint
lint:
	golangci-lint run

CONTROLLER_TOOLS_VERSION ?= v0.10.0
LOCALBIN ?= $(shell pwd)/.bin
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary. If wrong version is installed, it will be overwritten.
$(CONTROLLER_GEN): $(LOCALBIN)
	test -s $(LOCALBIN)/controller-gen && $(LOCALBIN)/controller-gen --version | grep -q $(CONTROLLER_TOOLS_VERSION) || \
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)

.PHONY: generate
generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object paths="./types/..."

$(LOCALBIN):
	mkdir -p $(LOCALBIN)

download-openapi-schemas:
	mkdir -p tmp

	# Canary Checker
	git clone --depth=1 git@github.com:flanksource/canary-checker.git tmp/canary-checker && cp tmp/canary-checker/config/schemas/* schema/openapi/

	# create schemas for specs only
	cat tmp/canary-checker/config/schemas/canary.schema.json | jq '.["$$ref"] = "#/definitions/CanarySpec"' > schema/openapi/canary.spec.schema.json
	cat tmp/canary-checker/config/schemas/component.schema.json | jq '.["$$ref"] = "#/definitions/ComponentSpec"' > schema/openapi/component.spec.schema.json
	cat tmp/canary-checker/config/schemas/system.schema.json | jq '.["$$ref"] = "#/definitions/SystemTemplateSpec"' > schema/openapi/system.spec.schema.json

	# Config DB
	git clone --depth=1 git@github.com:flanksource/config-db.git tmp/config-db && cp tmp/config-db/config/schemas/* schema/openapi/

	# create schemas for specs only
	cat tmp/config-db/config/schemas/scrape_config.schema.json | jq '.["$$ref"] = "#/definitions/ScrapeConfigSpec"' > schema/openapi/scrape_config.spec.schema.json

	# Cleanup
	rm -rf tmp
