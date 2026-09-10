// Package main serves the vendor license HTTP API for Telegram bots and operators.
//
// Role:
//   - Issue pilot and paid JWTs; expose SKU catalog and pending trial queue as Telegram button rows.
//   - Binds localhost by default; protect with LICENSE_VENDOR_API_TOKEN bearer auth.
//
// Verify:
// go run ./cmd/license-vendor-api --listen 127.0.0.1:8199
// curl -sf -H "Authorization: Bearer $LICENSE_VENDOR_API_TOKEN" http://127.0.0.1:8199/health
// OpenAPI: deploy/vendor/LICENSE_VENDOR_API.openapi.yaml
package main
