// Package metrics: Prometheus metric registration shared across binaries.
// Hot path: avoid per-request dynamic label values in filter loops.
//
package metrics
