.PHONY: build clean run run_iota run_iota_opa run_opa test

MICROSERVICES=cmd/calculator/calculator-go \
				cmd/populator/populator-go \
				cmd/populator-api/populator-api-go \
				cmd/subscriber/subscriber-go

.PHONY: $(MICROSERVICES)

VERSION=$(shell cat ./VERSION 2>/dev/null || echo 0.0.0)
DOCKER_TAG=$(VERSION)-dev

GIT_SHA=$(shell git rev-parse HEAD || echo 'v0.0.1')
GOFLAGS2=-ldflags "-X github.com/project-alvarium/scoring-apps-go.Version=$(GIT_SHA)"
GOTESTFLAGS?=-race

.PHONY: build
build: $(MICROSERVICES) ## Build all the service binaries

.PHONY: cmd/calculator/calculator-go
cmd/calculator/calculator-go:
	@echo "Building calculator-go"
	CGO_ENABLED=1 go build -o $@ ./cmd/calculator
	@echo "Finished calculator-go"

.PHONY: cmd/populator/populator-go
cmd/populator/populator-go:
	@echo "Building populator-go"
	go build -o $@ ./cmd/populator
	@echo "Finished populator-go"

.PHONY: cmd/populator-api/populator-api-go
cmd/populator-api/populator-api-go:
	@echo "Building populator-api-go"
	go build -o $@ ./cmd/populator-api
	@echo "Finished populator-api-go"

.PHONY: cmd/subscriber/subscriber-go
cmd/subscriber/subscriber-go:
	@echo "Building subscriber-go"
#	export CFLAGS=" -g -lm -ldl"
	CGO_ENABLED=1 go build -o $@ ./cmd/subscriber
	@echo "Finished subscriber-go"

.PHONY: run ## MQTT annotation pub/sub with local policy definition
run:
	cd scripts/bin && ./launch.sh

.PHONY: run_iota ## IOTA annotation pub/sub with local policy definition
run_iota:
	cd scripts/bin && ./launch_iota.sh

.PHONY: run_iota_opa ## IOTA annotation pub/sub with OPA server policy definition
run_iota_opa:
	cd scripts/bin && ./launch_iota_opa.sh

.PHONY: run_opa ## MQTT annotation pub/sub with OPA server policy definition
run_opa:
	cd scripts/bin && ./launch_opa.sh

.PHONY: clean
clean:
	@echo "Cleaning build artifacts"
	rm -f $(MICROSERVICES)
	rm -f coverage.out
	@echo "Done"

.PHONY: test
test: ## Runs the service unit tests, linter file formats, and verifies attribution files located
	@echo "About to test go services, execute tests etc."
	go test $(GOTESTFLAGS) -coverprofile=coverage.out ./...
	go vet ./...
	gofmt -l .
	[ "`gofmt -l .`" = "" ]
	@echo "Finished testing Go packages."