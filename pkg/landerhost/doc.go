// Package landerhost stores and serves hosted lander ZIP contents from a filesystem root.
//
// Role:
//   - Store: validate ZIP, extract under {root}/{lander_id}/v{N}, publish live symlink, serve static files.
//   - extractZipSafe + cleanRelativePath block zip-slip; ensureIndexHTML flattens single top-level folder.
//   - Editor helpers: ListVersionFiles, ReadVersionFile, WriteVersionTextFile, CloneVersion for draft versions.
//   - MintPreviewToken / VerifyPreviewToken: HMAC-SHA256 preview URLs scoped to lander_id + version + expiry.
//   - PublicURL and PreviewURL build tracker-facing paths (/lp/, /lp-preview/).
//
// Defaults and limits:
//   - DefaultMaxZipBytes 32 MiB per upload (ExtractZip and per-entry io.LimitReader).
//   - DefaultMaxEditorFileBytes 1 MiB per in-browser text edit.
//   - previewTokenTTL 1 hour from MintPreviewToken issue time.
//   - Directory mode 0o750; published files 0o640.
//
// Topology:
//   - Wired from internal/controlplane/flow_bridge.go (initLanderStore) and internal/flow hosted routes.
//   - Hosted editor handlers use pkg/coldpath.DecodeRequestOrBadRequest with DefaultMaxEditorFileBytes.
//   - stdlib archive/zip, os, path/filepath only; no Redis or Postgres in this package.
//
// Invariants:
//   - Zip slip blocked: cleanRelativePath rejects ..; pathWithinRoot checks every extract and open path.
//   - ExtractZip requires index.html at ZIP root or inside exactly one top-level folder (then flattened).
//   - PublishVersion atomically replaces live symlink via tmp + Rename; missing index.html fails closed.
//   - Preview token: constant-time signature compare (subtle.ConstantTimeCompare); expired or wrong lander_id rejected.
//   - IsEditableTextPath gates editor writes to .html, .css, .js, .txt, .json, .svg only.
//
// Zero-alloc / performance:
//   - Cold path only (admin upload, editor, static serve). File I/O and zip decode allocate; not on /track SLA.
//   - OpenLiveFile and OpenPreviewFile stream from disk; callers set Content-Length from FileInfo.
//
// Fail-closed:
//   - Traversal in ZIP entry name or editor path -> error, no partial extract left on disk (dest removed).
//   - VerifyPreviewToken returns ok=false on bad signature, expiry, or lander_id mismatch (no version leak).
//   - Nil Store or uuid.Nil lander_id -> error on mutating methods.
//   - Non-editable extension or oversize editor file -> error before write.
//
// Tradeoffs:
//   - Symlink live -> v{N} vs copying tree on publish: fast cutover, disk shared until next version.
//   - 32 MiB ZIP cap limits operator upload size; internal/flow adds 1 MiB headroom at HTTP boundary.
//   - Single-folder flatten heuristic accepts common bundler layout; multi-root zips without index.html fail.
//   - Preview token in query string: short TTL limits exposure; secret required from controlplane config.
//
// Forbidden:
//   - Import internal/* packages.
//   - Serve paths outside LiveDir or VersionDir roots (callers must use Store methods, not raw Join).
//
// Verify:
//
//	go test ./pkg/landerhost/... -short -run TestExtractZipSafe_holdoutPathTraversal -count=1
//	go test ./pkg/landerhost/... -short -run TestExtractZipPublish_roundTrip -count=1
//	go test ./pkg/landerhost/... -short -run TestPreviewToken_roundTrip -count=1
package landerhost
