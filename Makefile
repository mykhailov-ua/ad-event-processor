.PHONY: fmt fmt-check gen lint test test-fast test-unit test-integration test-fault test-int test-alloc-gate management-domain-coverage test-full test-resilience test-broker-fault-lab test-sentinel-resilience build build-bin release-build release-garble release-installer proto proto-grpc check-local pr-fast tier-a fraudtrain-check check-vuln bpf-dev bpf-session-start bpf-session-stop load-test-config load-test-bpf bpf-resource-gate bpf-nightly-gate cache-miss-gate escape-heap-gate cold-path-gates check-scripts-layout dev-preflight-smoke perf-gate-smoke edge-preflight openrtb-fuzz-smoke license-red-team license-pentest license-verify license-alloc-gate license-differential-gate license-red-team-garbled license-garbled-alloc-gate license-red-team-extended license-fuzz-nightly-gate release-qa-smoke license-gdb-guard-smoke release-strings-gate license-guard-test license-guard-off-smoke license-guard-fault-gate public-key-strings-gate asset-seal-salt-smoke hwid-strings-gate garble-literals-policy-gate garble-literals-p99-smoke bpf-edge-prereq-gate sealed-bpf-xdp-smoke clean-gitignored openapi-export openapi-types

BIN_DIR := bin
BIN_TAGS := timetzdata
BIN_LDFLAGS := -ldflags="-s -w -buildid="
RELEASE_LDFLAGS := -trimpath $(BIN_LDFLAGS)
RELEASE_GARBLE_CMDS := tracker processor control
PILOT_IMAGE_CMDS := tracker processor control
GARBLE_VERSION ?= v0.15.0

BIN_CMDS := tracker processor control ivt-detector fraud-scorer broker region-proxy log-shipper alertmanager-telegram log-evacuator log-compactor edge-xdp edge-bpf-sync

fmt:
	bash scripts/ci/format.sh

fmt-check:
	bash scripts/ci/format.sh --check

gen:
	bash scripts/ci/gen.sh --proto

openapi-export:
	go run ./cmd/openapi-export

.PHONY: openapi-types
openapi-types:
	@echo "openapi-types: skipped (web/ removed; regenerate TS types when admin_contract_gate ships)"

lint: gen fmt
	bash scripts/ci/lint.sh

test-fast: gen fmt
	go test -short -count=1 -timeout=240s -p=1 ./internal/... ./pkg/...

test-unit: test-fast

test-integration: gen fmt
	go test -count=1 -timeout 30m ./internal/... ./pkg/... ./tests/... -skip 'Fault'

test-fault: gen fmt
	go test -count=1 -timeout 30m -run 'Fault' ./...

test-alloc-gate: gen fmt
	go test -short -count=1 -run 'ZeroAlloc|zeroAlloc_fraudScoring|FraudScoring_LatencySLA|BrokerProducer|ApplyRtbAuction_shadow_zeroAlloc|RecordRtbShadow|HTTP1Parse|OpenRTB26_Exchange|Check_zeroAlloc_localQuantaFullSkip' ./internal/ingest/... ./internal/filter/... ./internal/stream/... ./internal/track/...
	go test -run='^$$' -bench='Benchmark(HTTP1Parse$$|TrackRequest_ParseJSONOpt$$|Auction$$|ParseOpenRTB26Split_hotOnly$$|RunOpenRTBExchangeParsed$$|TrackerToBroker$$|CIDR_LPM_Lookup_IPv4$$|CIDR_LPM_Lookup_IPv6$$|TLS_Fingerprint_|LinkSigner_Verify$$|ClickProxy_Stream$$)' -benchmem -count=1 ./internal/ingest/... ./internal/filter/... ./internal/stream/... ./internal/track/... ./internal/rtb/
	bash scripts/test/openrtb_fuzz_smoke.sh
	bash scripts/test/telegram/fuzz_smoke.sh
	bash scripts/test/cidr_fuzz_smoke.sh
	bash scripts/test/edge/click_fuzz_smoke.sh
	bash scripts/test/landing_protection_fuzz_smoke.sh

management-domain-coverage:
	bash scripts/ci/static/management_domain_coverage.sh

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

clean-gitignored:
	bash scripts/clean/gitignored.sh

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
	@ln -sf ad-event-processor-install $(BIN_DIR)/ad-event-processor-install 2>/dev/null || cp -f $(BIN_DIR)/ad-event-processor-install $(BIN_DIR)/ad-event-processor-install

RELEASE_PLATFORMS := linux/amd64 linux/arm64
RELEASE_CMDS := tracker processor control ivt-detector fraud-scorer region-proxy broker

release-build: gen fmt
	@mkdir -p $(BIN_DIR)
	@set -e; \
	for platform in $(RELEASE_PLATFORMS); do \
	  GOOS=$${platform%/*}; GOARCH=$${platform
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
	  GOOS=$${platform%/*}; GOARCH=$${platform
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
	bash scripts/test/license/red_team_extended.sh

license-pentest:
	bash scripts/security/license_pentest.sh

license-verify:
	bash scripts/ci/license/verify_tier.sh

license-alloc-gate:
	bash scripts/ci/license/alloc.sh

license-differential-gate:
	bash scripts/ci/license/differential.sh

asset-seal-salt-smoke:
	bash scripts/ci/license/asset_seal_salt_smoke.sh

hwid-strings-gate:
	bash scripts/ci/license/hwid_strings.sh

license-guard-test:
	go test -tags=license_guard ./internal/licensing/ -run Guard -count=1

license-guard-off-smoke:
	bash scripts/test/license/guard_off_smoke.sh

license-guard-fault-gate:
	bash scripts/ci/license/guard_fault.sh

public-key-strings-gate:
	bash scripts/ci/license/public_key_strings.sh

license-red-team-garbled:
	bash scripts/ci/license/red_team_garbled.sh

license-garbled-alloc-gate:
	bash scripts/ci/license/garbled_alloc.sh

license-red-team-extended:
	bash scripts/test/license/red_team_extended.sh

license-fuzz-nightly-gate:
	bash scripts/ci/license/fuzz_nightly.sh

release-qa-smoke:
	bash scripts/test/release/qa_smoke.sh

license-gdb-guard-smoke:
	bash scripts/test/license/gdb_guard_smoke.sh

release-strings-gate:
	bash scripts/ci/license/release_strings.sh \
		$(BIN_DIR)/garbled-release/tracker \
		$(BIN_DIR)/garbled-release/processor \
		$(BIN_DIR)/garbled-release/control

garble-literals-eval:
	bash scripts/ci/license/garble_literals_eval.sh

garble-literals-policy-gate:
	bash scripts/ci/license/garble_literals_policy.sh

garble-literals-p99-smoke:
	bash scripts/test/license/garble_literals_p99_smoke.sh

bpf-edge-prereq-gate:
	bash scripts/ci/bpf/edge_prereq.sh

sealed-bpf-xdp-smoke:
	bash scripts/test/edge/sealed_bpf_xdp_smoke.sh

bpf-dev:
	bash scripts/dev/stack/bpf_setup.sh

bpf-session-start: bpf-dev
	sudo bash scripts/dev/stack/bpf_session.sh start

bpf-session-stop:
	sudo bash scripts/dev/stack/bpf_session.sh stop

load-test-config:
	bash scripts/lib/render_load_test_config.sh

load-test-bpf: bpf-dev load-test-config
	AD_EVENT_PROCESSOR_BPF_PROBE=1 AD_EVENT_PROCESSOR_BPF_SAMPLE_RATE=$${AD_EVENT_PROCESSOR_BPF_SAMPLE_RATE:-10} bash scripts/test/load/malformed.sh business

bpf-resource-gate:
	BPF_GATE_STRICT=true bash scripts/ci/bpf/resource.sh

bpf-nightly-gate:
	BPF_GATE_STRICT=true bash scripts/test/bpf/nightly_job.sh hot
	BPF_GATE_STRICT=true bash scripts/test/bpf/nightly_job.sh cold

cache-miss-gate:
	bash scripts/perf/cache_miss_nightly_job.sh

cold-path-gates:
	bash scripts/ci/static/anti_slop.sh
	bash scripts/ci/static/diff_assertion.sh
	bash scripts/ci/static/sql_safety.sh
	bash scripts/ci/static/hot_path_static.sh
	bash scripts/ci/static/cold_path_static.sh

escape-heap-gate:
	bash scripts/ci/static/escape_heap.sh

check-scripts-layout:
	bash scripts/ci/naming/scripts_layout.sh

dev-preflight-smoke:
	bash scripts/dev/stack/preflight.sh

seed-admin:
	bash scripts/dev/stack/seed_admin.sh

perf-gate-smoke:
	PERF_GATE_STRICT=false bash scripts/test/load/gate_run.sh

openrtb-fuzz-smoke:
	bash scripts/test/openrtb_fuzz_smoke.sh

telegram-fuzz-smoke:
	bash scripts/test/telegram/fuzz_smoke.sh

tg-hotpath-soak:
	bash scripts/test/telegram/hotpath_soak.sh

telegram-hotpath-gate:
	bash scripts/test/telegram/hotpath_gate.sh

edge-preflight:
	bash scripts/ops/edge_preflight.sh

proto:
	bash scripts/ci/gen.sh --proto

proto-grpc:
	bash scripts/ci/gen.sh --proto
