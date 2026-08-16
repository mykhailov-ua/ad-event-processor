.PHONY: fmt gen lint test test-fast test-unit test-integration test-fault test-int test-alloc-gate management-domain-coverage test-full test-resilience test-broker-fault-lab test-sentinel-resilience build build-bin release-build release-garble release-installer proto proto-grpc check-local pr-fast tier-a fraudtrain-check check-vuln bpf-dev bpf-session-start bpf-session-stop load-test-bpf check-scripts-layout dev-preflight-smoke perf-gate-smoke edge-phase0 openrtb-fuzz-smoke license-red-team license-verify license-alloc-gate license-differential-gate license-red-team-garbled license-garbled-alloc-gate license-red-team-extended license-gdb-guard-smoke release-strings-gate license-guard-test license-guard-off-smoke license-guard-fault-gate public-key-strings-gate asset-seal-salt-smoke hwid-strings-gate garble-literals-policy-gate garble-literals-p99-smoke bpf-edge-prereq-gate sealed-bpf-xdp-smoke

BIN_DIR := bin
BIN_TAGS := timetzdata
BIN_LDFLAGS := -ldflags="-s -w -buildid="
RELEASE_LDFLAGS := -trimpath $(BIN_LDFLAGS)
RELEASE_GARBLE_CMDS := tracker processor control
PILOT_IMAGE_CMDS := tracker processor control
GARBLE_VERSION ?= v0.15.0
# Match deploy/docker/Dockerfile multi-binary image + local installer CLI.
BIN_CMDS := tracker processor control ivt-detector fraud-scorer broker region-proxy log-shipper alertmanager-telegram log-evacuator log-compactor edge-xdp edge-bpf-sync

fmt:
	go fmt ./...

gen:
	bash scripts/ci/gen.sh --proto

lint: gen fmt
	bash scripts/ci/lint_go_gate.sh all

test-fast: gen fmt
	go test -short -count=1 -timeout=120s ./internal/... ./pkg/...

test-unit: test-fast

test-integration: gen fmt
	go test -count=1 -timeout 30m ./internal/... ./pkg/... ./tests/... -skip 'Fault'

test-fault: gen fmt
	go test -count=1 -timeout 30m -run 'Fault' ./...

test-alloc-gate: gen fmt
	go test -short -count=1 -run 'ZeroAlloc|zeroAlloc_fraudScoring|FraudScoring_LatencySLA|BrokerProducer|ApplyRtbAuction_shadow_zeroAlloc|RecordRtbShadow|HTTP1Parse|OpenRTB26_Exchange|Check_zeroAlloc_localQuantaFullSkip' ./internal/ingestion/...
	go test -run='^$$' -bench='Benchmark(HTTP1Parse$$|TrackRequest_ParseJSONOpt$$|Auction$$|ParseOpenRTB26Split_hotOnly$$|RunOpenRTBExchangeParsed$$|TrackerToBroker$$|CIDR_LPM_Lookup_IPv4$$|CIDR_LPM_Lookup_IPv6$$|ClickProxy_Stream$$)' -benchmem -count=1 ./internal/ingestion/... ./internal/rtb/
	bash scripts/test/openrtb_fuzz_smoke.sh
	bash scripts/test/telegram_fuzz_smoke.sh
	bash scripts/test/cidr_fuzz_smoke.sh
	bash scripts/test/click_proxy_fuzz_smoke.sh

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
	@echo "build-bin: ad-event-processor-install -> $(BIN_DIR)/ad-event-processor-install"
	@CGO_ENABLED=0 go build -tags $(BIN_TAGS) $(BIN_LDFLAGS) -o $(BIN_DIR)/ad-event-processor-install ./cmd/installer
	@ln -sf ad-event-processor-install $(BIN_DIR)/espx-install 2>/dev/null || cp -f $(BIN_DIR)/ad-event-processor-install $(BIN_DIR)/espx-install

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

license-verify:
	bash scripts/ci/license_verify_tier.sh

license-alloc-gate:
	bash scripts/ci/license_alloc_gate.sh

license-differential-gate:
	bash scripts/ci/license_differential_gate.sh

asset-seal-salt-smoke:
	bash scripts/ci/asset_seal_salt_smoke.sh

hwid-strings-gate:
	bash scripts/ci/hwid_strings_gate.sh

license-guard-test:
	go test -tags=license_guard ./internal/licensing/ -run Guard -count=1

license-guard-off-smoke:
	bash scripts/test/license_guard_off_smoke.sh

license-guard-fault-gate:
	bash scripts/ci/license_guard_fault_gate.sh

public-key-strings-gate:
	bash scripts/ci/public_key_strings_gate.sh

license-red-team-garbled:
	bash scripts/ci/license_red_team_garbled.sh

license-garbled-alloc-gate:
	bash scripts/ci/license_garbled_alloc_gate.sh

license-red-team-extended:
	bash scripts/test/license_red_team_extended.sh

license-gdb-guard-smoke:
	bash scripts/test/license_gdb_guard_smoke.sh

release-strings-gate:
	bash scripts/ci/release_strings_gate.sh $(BIN_DIR)/tracker

garble-literals-eval:
	bash scripts/ci/garble_literals_eval.sh

garble-literals-policy-gate:
	bash scripts/ci/garble_literals_policy_gate.sh

garble-literals-p99-smoke:
	bash scripts/test/garble_literals_p99_smoke.sh

bpf-edge-prereq-gate:
	bash scripts/ci/bpf_edge_prereq_gate.sh

sealed-bpf-xdp-smoke:
	bash scripts/test/sealed_bpf_xdp_smoke.sh

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
