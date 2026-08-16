# Shared hot-path package roots for Go lint gates.
# Keep aligned with .cursor/rules/hot-path.mdc globs + internal/rtb (alloc-gate auction).
lint_go_hot_path_dirs=(
	internal/ingestion
	internal/domain
	internal/rtb
	cmd/tracker
	pkg/broker
)

# golangci-lint --skip-dirs regex (pipe-separated).
lint_go_hot_path_skip_re='internal/ingestion|internal/domain|internal/rtb|cmd/tracker|pkg/broker'

# Per-request /track and auction sources only (not workers, stores, health, corpus fixtures).
lint_go_hot_path_request_files=(
	internal/ingestion/track_core.go
	internal/ingestion/track_ingest_gnet.go
	internal/ingestion/track_cors.go
	internal/ingestion/track_cors_http.go
	internal/ingestion/track_cors_handler.go
	internal/ingestion/handler_http1_fsm.go
	internal/ingestion/handler_http1_chunked.go
	internal/ingestion/handler_http1_validate.go
	internal/ingestion/handler_http1_idle.go
	internal/ingestion/handler_http2.go
	internal/ingestion/requests_parse.go
	internal/ingestion/openrtb_parse.go
	internal/ingestion/openrtb_26_window_scan.go
	internal/rtb/auction.go
	internal/rtb/auction_rank.go
	internal/rtb/auction_clearing.go
	internal/rtb/auction_ranking.go
	internal/rtb/no_bid.go
	internal/rtb/vast_decode.go
	internal/rtb/vast_parse.go
)
