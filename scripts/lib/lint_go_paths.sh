# shellcheck shell=bash

# Role: Library: Go path lint helper.
# Execution context: Sourced by CI, fault, and dev scripts; not a standalone gate.
# Invariants/contracts enforced: Helpers must not exit 0 on error paths when used as gate prerequisites.
# Verify: source scripts/lib/lint_go_paths.sh
lint_go_hot_path_dirs=(
  internal/ingest
  internal/filter
  internal/track
  internal/stream
  internal/domain
  internal/rtb
  cmd/tracker
  pkg/broker
)

lint_go_hot_path_skip_re='internal/ingest|internal/filter|internal/track|internal/stream|internal/domain|internal/rtb|cmd/tracker|pkg/broker'

lint_go_hot_path_request_files=(
  internal/ingest/track_core.go
  internal/ingest/track_ingest_gnet.go
  internal/ingest/handler_http1_fsm.go
  internal/ingest/handler_http1_chunked.go
  internal/ingest/handler_http1_validate.go
  internal/ingest/handler_http1_idle.go
  internal/ingest/handler_http2.go
  internal/ingest/requests_parse.go
  internal/ingest/openrtb_parse.go
  internal/ingest/openrtb_26_window_scan.go
  internal/rtb/auction.go
  internal/rtb/auction_rank.go
  internal/rtb/auction_clearing.go
  internal/rtb/auction_ranking.go
  internal/rtb/no_bid.go
  internal/rtb/vast_decode.go
  internal/rtb/vast_parse.go
)
