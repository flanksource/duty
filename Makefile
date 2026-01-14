## Tool Binaries
LOCALBIN ?= $(shell pwd)/.bin
GOLANGCI_LINT ?= $(LOCALBIN)/golangci-lint
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen

## Tool Versions
CONTROLLER_TOOLS_VERSION ?= v0.19.0
GOLANGCI_LINT_VERSION ?= v2.7.2

.PHONY: ginkgo
ginkgo:
	go install github.com/onsi/ginkgo/v2/ginkgo

test: ginkgo
	ginkgo -r -v --skip-package=tests/e2e

.PHONY: test-e2e
test-e2e: ginkgo
	cd tests/e2e && docker-compose up -d && \
	timeout 60 bash -c 'until curl -s http://localhost:3100/ready >/dev/null 2>&1; do sleep 2; done' && \
	(ginkgo -v; TEST_EXIT_CODE=$$?; docker-compose down; exit $$TEST_EXIT_CODE)

.PHONY: e2e-services
e2e-services: ## Run e2e test services in foreground with automatic cleanup on exit
	cd tests/e2e && \
	trap 'docker-compose down -v && docker-compose rm -f' EXIT INT TERM && \
	docker-compose up --remove-orphans

.PHONY: bench
bench:
	go test -bench=. -benchtime=10s -timeout 30m github.com/flanksource/duty/bench

.PHONY: bench-ci
bench-ci:
	DUTY_BENCH_SIZES=10000,25000 go test -bench=. -benchtime=1s -count=1 -timeout 30m github.com/flanksource/duty/bench

fmt:
	go fmt ./...

.PHONY: lint
lint: golangci-lint
	$(GOLANGCI_LINT) run ./...
	go vet ./...

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary. If wrong version is installed, it will be overwritten.
$(CONTROLLER_GEN): $(LOCALBIN)
	test -s $(LOCALBIN)/controller-gen && $(LOCALBIN)/controller-gen --version | grep -q $(CONTROLLER_TOOLS_VERSION) || \
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)

# Generate OpenAPI schema
.PHONY: gen-schemas
gen-schemas:
	cp go.mod hack/generate-schemas && \
	cd hack/generate-schemas && \
	go mod edit -module=github.com/flanksource/duty/hack/generate-schemas && \
	go mod edit -replace=github.com/flanksource/duty=../../../duty && \
	go mod tidy && \
	go run ./main.go

.PHONY: generate
generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object paths="./types/..."
	$(CONTROLLER_GEN) object paths="./logs/..."
	$(CONTROLLER_GEN) object paths="./connection/..."
	$(CONTROLLER_GEN) object paths="./models/..."
	$(CONTROLLER_GEN) object paths="./shell/..."
	$(CONTROLLER_GEN) object paths="./pubsub/..."
	$(CONTROLLER_GEN) object paths="./dataquery/..."
	$(CONTROLLER_GEN) object paths="./view/..."
	$(CONTROLLER_GEN) object paths="./"
	PATH=$(LOCALBIN):${PATH} go generate ./...

.PHONY: manifests
manifests: generate gen-schemas

$(LOCALBIN):
	mkdir -p $(LOCALBIN)

update-schemas: download-openapi-schemas schema-definitions

schema-definitions:
	cat schema/openapi/connection.schema.json |  jq 'del(.["$$ref"], .["$$id"], .["$$schema"] )' > schema/openapi/connection.definitions.json
	cat schema/openapi/scrape_config.spec.schema.json |  jq 'del(.["$$ref"], .["$$id"], .["$$schema"] )' > schema/openapi/scrape_config.definitions.json
	cat schema/openapi/notification.schema.json |  jq 'del(.["$$ref"], .["$$id"], .["$$schema"] )' > schema/openapi/notification.definitions.json
	cat schema/openapi/topology.spec.schema.json |  jq 'del(.["$$ref"], .["$$id"], .["$$schema"] )' > schema/openapi/topology.definitions.json
	cat schema/openapi/playbook-spec.schema.json |  jq 'del(.["$$ref"], .["$$id"], .["$$schema"] )' > schema/openapi/playbook.definitions.json


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


hack/migrate/go.mod: tidy
	cp go.mod hack/migrate && \
	cd hack/migrate && \
	go mod edit -module=github.com/flanksource/duty/hack/migrate && \
	go mod edit -require=github.com/flanksource/duty@v1.0.0 && \
 	go mod edit -replace=github.com/flanksource/duty=../../ && \
	go mod tidy

.PHONY: migrate-test
migrate-test: hack/migrate/go.mod
	cd hack/migrate && go run ./main.go

cp-mission-control-openapi-schemas:
	cp ../mission-control/config/schemas/*.json schema/openapi/

fmt_json:
	ls fixtures/expectations/*.json | while read -r jf; do \
		cat <<< $$(jq . $$jf) > $$jf; \
	done;

fmt_sql:
	ls views/*.sql | while read -r sqlf; do \
		pg_format -s 2 -o $$sqlf $$sqlf; \
	done;

tidy:
	go mod tidy

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT) ## Download golangci-lint locally if necessary.
$(GOLANGCI_LINT): $(LOCALBIN)
	test -s $(LOCALBIN)/golangci-lint || curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b $(LOCALBIN) $(GOLANGCI_LINT_VERSION)
