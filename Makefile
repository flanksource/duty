test:
	go test ./... -test.v

fmt:
	go fmt ./...

.PHONY: lint
lint:
	golangci-lint run

CONTROLLER_TOOLS_VERSION ?= v0.14.0
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
	$(CONTROLLER_GEN) object paths="./connection/..."
	$(CONTROLLER_GEN) object paths="./models/..."

$(LOCALBIN):
	mkdir -p $(LOCALBIN)

download-openapi-schemas:
	mkdir -p tmp

	# Canary Checker
	git clone --depth=1 git@github.com:flanksource/canary-checker.git tmp/canary-checker && cp tmp/canary-checker/config/schemas/* schema/openapi/

	# create schemas for specs only
	cat tmp/canary-checker/config/schemas/canary.schema.json | jq '.["$$ref"] = "#/$$defs/CanarySpec"' > schema/openapi/canary.spec.schema.json
	cat tmp/canary-checker/config/schemas/component.schema.json | jq '.["$$ref"] = "#/$$defs/ComponentSpec"' > schema/openapi/component.spec.schema.json
	cat tmp/canary-checker/config/schemas/topology.schema.json | jq '.["$$ref"] = "#/$$defs/TopologySpec"' > schema/openapi/topology.spec.schema.json

	# Config DB
	git clone --depth=1 git@github.com:flanksource/config-db.git tmp/config-db && cp tmp/config-db/config/schemas/* schema/openapi/

	# APM-Hub
	git clone --depth=1 git@github.com:flanksource/apm-hub.git tmp/apm-hub && cp tmp/apm-hub/config/schemas/* schema/openapi/

	# Mission control
	git clone --depth=1 git@github.com:flanksource/mission-control.git tmp/mission-control && cp tmp/mission-control/config/schemas/* schema/openapi/

	# create schemas for specs only
	cat tmp/config-db/config/schemas/scrape_config.schema.json | jq '.["$$ref"] = "#/definitions/ScraperSpec"' > schema/openapi/scrape_config.spec.schema.json

	# Cleanup
	rm -rf tmp


cp-playbook-schema:
	cp ../incident-commander/config/schemas/playbook-spec.schema.json schema/

fmt_json:
	ls fixtures/expectations/*.json | while read -r jf; do \
		cat <<< $$(jq . $$jf) > $$jf; \
	done;

fmt_sql:
	ls views/*.sql | while read -r sqlf; do \
		sql-formatter -l postgresql --fix $$sqlf; \
	done;

tidy:
	go mod tidy
	cd hack/migrate && go mod tidy
