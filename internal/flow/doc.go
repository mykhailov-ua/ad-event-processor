// Package flow owns lander/offer/flow CRUD, path validation, hosted lander ZIP serve, and MAB bandit updates.
//
// Role:
//   - Admin HTTP: /api/v1/landers, /api/v1/offers, /api/v1/flows (CRUD), hosted-upload, hosted-editor routes.
//   - Public hosted lander serve: GET /lp/{lander_id}/{path...} (ServeHostedLanderFile).
//   - validate.go enforces path shape before publish; bandit_tx.go applies Thompson/proportional weight updates from campaign worker ticks.
//   - Hosted lander assets stored on disk; editor draft/publish under /api/v1/landers/{id}/hosted-*.
//
// Topology:
//   - Wired from controlplane via flow bridge; Host port supplies lander base URL, path validation, and campaign reload pub/sub.
//   - CampaignFlowHost publishes campaign updates after flow mutations affecting routing.
//   - Bandit state in Postgres; tracker reads weights from Redis/catalog snapshots only.
//
// Invariants:
//   - Flow path refs must resolve (URL or hosted_asset_id) before campaign publish gate passes.
//   - Bandit apply is transactional; partial weight updates roll back on PG error.
//   - Hosted ZIP uploads respect landerhost.DefaultMaxZipBytes + 1 MiB headroom at handler boundary.
//
// Forbidden:
//   - Postgres or bandit math on tracker /track hot path.
//   - Import internal/controlplane admin handlers from this package.
//
// Verify:
// go test ./internal/flow/ -short -count=1
// go test ./internal/flow/ -short -run TestValidatePathShape -count=1
// go test ./internal/flow/ -short -run TestBandit_WorkerUpdatesWeights -count=1
package flow
