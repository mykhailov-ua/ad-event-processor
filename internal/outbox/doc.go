// Package outbox implements shared outbox polling primitives and protobuf payload codecs.
// The OutboxWorker that applies Redis side effects remains in controlplane until domain handlers move out.
package outbox
