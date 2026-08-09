.PHONY: fmt gen lint test test-fast test-unit test-integration test-fault test-int test-alloc-gate management-domain-coverage test-full test-resilience test-broker-fault-lab test-sentinel-resilience build build-bin release-build release-garble release-installer proto proto-grpc check-local pr-fast tier-a fraudtrain-check check-vuln bpf-dev bpf-session-start bpf-session-stop load-test-bpf check-scripts-layout dev-preflight-smoke perf-gate-smoke edge-phase0 openrtb-fuzz-smoke

BIN_DIR := bin
BIN_TAGS := timetzdata
BIN_LDFLAGS := -ldflags="-s -w -buildid="
RELEASE_LDFLAGS := -trimpath $(BIN_LDFLAGS)
RELEASE_GARBLE_CMDS := tracker processor control
PILOT_IMAGE_CMDS := tracker processor control
GARBLE_VERSION ?= v0.14.2
# Match deploy/docker/Dockerfile multi-binary image + local installer CLI.
BIN_CMDS := tracker processor control ivt-detector fraud-scorer broker region-proxy log-shipper alertmanager-telegram log-evacuator log-compactor edge-xdp edge-bpf-sync

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
	go test -short -count=1 -timeout=120s ./internal/... ./pkg/...

test-unit: test-fast

test-integration: gen fmt
	go test -count=1 -timeout 30m ./internal/... ./pkg/... ./tests/... -skip 'Fault'

test-fault: gen fmt
	go test -count=1 -timeout 30m -run 'Fault' ./...

test-alloc-gate: gen fmt
	go test -short -count=1 -run 'ZeroAlloc|zeroAlloc_fraudScoring|FraudScoring_LatencySLA|BrokerProducer|ApplyRtbAuction_shadow_zeroAlloc|RecordRtbShadow|HTTP1Parse|OpenRTB26_Exchange|Check_zeroAlloc_localQuantaFullSkip' ./internal/ingestion/...
	go test -run='^$$' -bench='Benchmark(HTTP1Parse$$|TrackRequest_ParseJSONOpt$$|Auction$$|ParseOpenRTB26Split_hotOnly$$|RunOpenRTBExchangeParsed$$|TrackerToBroker$$)' -benchmem -count=1 ./internal/ingestion/... ./internal/rtb/
	bash scripts/test/openrtb_fuzz_smoke.sh
	bash scripts/test/telegram_fuzz_smoke.sh

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

pr-fast:
	bash scripts/ci/pr_fast.sh

fraudtrain-check:
	bash scripts/ci/fraudtrain.sh

tier-a:
	bash scripts/ci/tier_a.sh

check-vuln:
	bash scripts/ci/govulncheck.sh

build: gen fmt
	docker build -t ad-event-processor:latest .

build-bin: gen fmt
	@mkdir -p $(BIN_DIR)
	@set -e; \
	for cmd in $(BIN_CMDS); do \
	  echo "build-bin: $$cmd -> $(BIN_DIR)/$$cmd"; \
	  CGO_ENABLED=0 go build -tags $(BIN_TAGS) $(BIN_LDFLAGS) -o $(BIN_DIR)/$$cmd ./cmd/$$cmd; \
	done
	@echo "build-bin: espx-install -> $(BIN_DIR)/espx-install"
	@CGO_ENABLED=0 go build -tags $(BIN_TAGS) $(BIN_LDFLAGS) -o $(BIN_DIR)/espx-install ./cmd/installer

RELEASE_PLATFORMS := linux/amd64 linux/arm64
RELEASE_CMDS := tracker processor control ivt-detector fraud-scorer region-proxy broker

release-build: gen fmt
	@mkdir -p $(BIN_DIR)
	@set -e; \
	for platform in $(RELEASE_PLATFORMS); do \
	  GOOS=$${platform%/*}; GOARCH=$${platform#*/}; \
	  for cmd in $(RELEASE_GARBLE_CMDS); do \
	    echo "release-build: $$cmd $$GOOS/$$GOARCH -> $(BIN_DIR)/$${cmd}-$${GOOS}-$${GOARCH}"; \
	    CGO_ENABLED=0 GOOS=$$GOOS GOARCH=$$GOARCH go build -tags $(BIN_TAGS) $(RELEASE_LDFLAGS) \
	      -o $(BIN_DIR)/$${cmd}-$${GOOS}-$${GOARCH} ./cmd/$${cmd}; \
	  done; \
	done

release-garble: gen fmt
	@chmod +x scripts/ci/release_garble.sh
	@RELEASE_GARBLE=1 bash scripts/ci/release_garble.sh $(BIN_DIR) $(RELEASE_GARBLE_CMDS)

release-garble-all-platforms: gen fmt
	@chmod +x scripts/ci/release_garble.sh
	@set -e; \
	for platform in $(RELEASE_PLATFORMS); do \
	  GOOS=$${platform%/*}; GOARCH=$${platform#*/}; \
	  out="$(BIN_DIR)/release-$${GOOS}-$${GOARCH}"; \
	  mkdir -p "$$out"; \
	  echo "release-garble-all-platforms: $$GOOS/$$GOARCH -> $$out"; \
	  RELEASE_GARBLE=1 GOOS=$$GOOS GOARCH=$$GOARCH bash scripts/ci/release_garble.sh "$$out" $(RELEASE_GARBLE_CMDS); \
	done

release-installer:
	bash scripts/install/release_pack.sh $(if $(VERSION),$(VERSION),)

license-issue:
	go run ./cmd/license-issue $(ARGS)

license-red-team:
	bash scripts/security/license_red_team.sh

garble-literals-eval:
	bash scripts/ci/garble_literals_eval.sh

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

seed-admin:
	bash scripts/dev/seed_admin.sh

perf-gate-smoke:
	PERF_GATE_STRICT=false bash scripts/test/gate_run.sh

openrtb-fuzz-smoke:
	bash scripts/test/openrtb_fuzz_smoke.sh

telegram-fuzz-smoke:
	bash scripts/test/telegram_fuzz_smoke.sh

tg-hotpath-soak:
	bash scripts/test/tg_hotpath_soak.sh

telegram-hotpath-gate:
	bash scripts/test/telegram_hotpath_gate.sh

edge-phase0:
	bash scripts/ops/phase0.sh

proto:
	bash scripts/ci/gen.sh --proto

proto-grpc:
	bash scripts/ci/gen.sh --proto
