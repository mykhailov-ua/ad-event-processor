// Package fraud: cold-path ML scoring, feature reads, training hooks. Production infer
// runs in cmd/fraud-scorer and cmd/ivt-detector sidecars.
//
// Hard rule: internal/ingestion must NOT import this package for scoring on /track.
//
package fraud
