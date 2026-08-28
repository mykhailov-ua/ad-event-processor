// Package outbox: shared outbox event helpers and types used across control-plane
// workers. Authoritative apply path is controlplane OutboxWorker + PG outbox_events.
//
package outbox
