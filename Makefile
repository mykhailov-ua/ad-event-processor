.PHONY: fmt gen lint test test-fast test-unit test-integration test-fault test-int test-alloc-gate management-domain-coverage test-full test-resilience test-broker-fault-lab test-sentinel-resilience build proto check-local check-vuln bpf-dev bpf-session-start bpf-session-stop load-test-bpf openapi-lint openapi-gen

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
	bash scripts/fault/run.sh

test-broker-fault-lab:
	bash scripts/fault/broker_fault_lab.sh

test-sentinel-resilience:
	bash scripts/fault/sentinel.sh

test: test-fast test-int

test-full: fmt
	bash scripts/ci/full_test.sh

check-local:
	bash scripts/ci/local_check.sh

check-vuln:
	bash scripts/ci/govulncheck.sh

openapi-lint:
	bash scripts/ci/openapi.sh

openapi-gen:
	go run ./cmd/openapi-gen

build: gen fmt
	docker build -t ad-event-processor:latest .

bpf-dev:
	bash scripts/dev/bpf_setup.sh

bpf-session-start: bpf-dev
	sudo bash scripts/dev/bpf_session.sh start

bpf-session-stop:
	sudo bash scripts/dev/bpf_session.sh stop

load-test-bpf: bpf-dev
	sudo ESPX_BPF_PROBE=1 ESPX_BPF_SAMPLE_RATE=$${ESPX_BPF_SAMPLE_RATE:-10} bash scripts/load/malformed.sh business

proto:
	bash scripts/ci/gen.sh --proto
