.PHONY: fmt gen lint test test-fast test-unit test-integration test-fault test-int test-alloc-gate management-domain-coverage test-full test-resilience test-broker-fault-lab test-sentinel-resilience build release-build proto proto-grpc check-local tier-a fraudtrain-check check-vuln bpf-dev bpf-session-start bpf-session-stop load-test-bpf openapi-lint openapi-gen check-scripts-layout dev-preflight-smoke perf-gate-smoke edge-phase0

fmt:
	go fmt ./...

gen:
	bash scripts/ci/gen.sh --proto

lint: gen fmt
	@if [ -z "$$(which golangci-lint 2> /dev/null)" ]; then \
		echo "Installing golangci-lint..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.5; \
	fi
	@GOPATH=$$(go env GOPATH); \
	if [ -z "$$GOPATH" ]; then GOPATH=$$HOME/go; fi; \
	$$GOPATH/bin/golangci-lint run

test-fast: gen fmt
	go test -short -count=1 ./internal/... ./pkg/...

test-unit: test-fast

test-integration: gen fmt
	go test -count=1 -timeout 30m ./internal/... ./pkg/... ./tests/... -skip 'Fault'

test-fault: gen fmt
	go test -count=1 -timeout 30m -run 'Fault' ./...

test-alloc-gate: gen fmt
	go test -short -count=1 -run 'ZeroAlloc|zeroAlloc_fraudScoring|FraudScoring_LatencySLA|ApplyRtbAuction_shadow_zeroAlloc|RecordRtbShadow|HTTP1Parse' ./internal/ingestion/...
	go test -run='^$$' -bench='Benchmark(HTTP1Parse$$|TrackRequest_ParseJSONOpt$$|Auction$$)' -benchmem -count=1 ./internal/ingestion/... ./internal/rtb/

management-domain-coverage:
	bash scripts/ci/management_domain_coverage.sh

test-int: gen fmt
	go test -v ./tests/...

test-resilience:
	bash scripts/test/run_resilience.sh

test-broker-fault-lab:
	bash scripts/test/broker_fault_lab.sh

test-sentinel-resilience:
	bash scripts/test/sentinel.sh

test: test-fast test-int

test-full: fmt
	bash scripts/ci/full_test.sh

check-local:
	bash scripts/ci/local_check.sh

fraudtrain-check:
	bash scripts/ci/fraudtrain.sh

tier-a:
	bash scripts/ci/tier_a.sh

check-vuln:
	bash scripts/ci/govulncheck.sh

openapi-lint:
	bash scripts/ci/openapi.sh

openapi-gen:
	go run ./cmd/openapi-gen

build: gen fmt
	docker build -t ad-event-processor:latest .

# Stripped linux/amd64 + linux/arm64 service binaries → dist/release/ (GAP-PROD-10 / P44).
# Not run in CI; vendor release pipeline or local smoke before shipping Pro builds.
RELEASE_DIR := dist/release
RELEASE_LDFLAGS := -ldflags="-s -w"
RELEASE_PLATFORMS := linux/amd64 linux/arm64
RELEASE_CMDS := tracker processor control ivt-detector fraud-scorer region-proxy broker

release-build: gen fmt
	@mkdir -p $(RELEASE_DIR)
	@set -e; \
	for platform in $(RELEASE_PLATFORMS); do \
	  GOOS=$${platform%/*}; GOARCH=$${platform#*/}; \
	  for cmd in $(RELEASE_CMDS); do \
	    echo "release-build: $$cmd $$GOOS/$$GOARCH"; \
	    CGO_ENABLED=0 GOOS=$$GOOS GOARCH=$$GOARCH go build -tags timetzdata $(RELEASE_LDFLAGS) \
	      -o $(RELEASE_DIR)/$${cmd}-$${GOOS}-$${GOARCH} ./cmd/$${cmd}; \
	  done; \
	done

bpf-dev:
	bash scripts/dev/bpf_setup.sh

bpf-session-start: bpf-dev
	sudo bash scripts/dev/bpf_session.sh start

bpf-session-stop:
	sudo bash scripts/dev/bpf_session.sh stop

load-test-bpf: bpf-dev
	sudo ESPX_BPF_PROBE=1 ESPX_BPF_SAMPLE_RATE=$${ESPX_BPF_SAMPLE_RATE:-10} bash scripts/test/malformed.sh business

check-scripts-layout:
	bash scripts/ci/check_scripts_layout.sh

dev-preflight-smoke:
	bash scripts/dev/preflight.sh

perf-gate-smoke:
	PERF_GATE_STRICT=false bash scripts/test/gate_run.sh

edge-phase0:
	bash scripts/ops/phase0.sh

proto:
	bash scripts/ci/gen.sh --proto

proto-grpc:
	bash scripts/ci/gen.sh --proto
