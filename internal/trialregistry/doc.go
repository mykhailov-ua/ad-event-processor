// Package trialregistry stores vendor-plane trial anchors for pilot JWT issue.
//
// File-backed registry (default deploy/vendor/trial_registry.json). Lives on the
// vendor workstation only; not shipped in buyer appliance Postgres migrations.
package trialregistry
