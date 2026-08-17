// Package domain implements domain support for BidShard.
package domain

import "time"

type BrokerConsumerConfig struct {
	BrokerAddr string
	RedisURL   string
	Topic      string
	Partition  uint16
	Group      string
	BatchSize  int
	FlushInt   time.Duration
	MaxBytes   uint32
	Timeout    time.Duration
	IdleWait   time.Duration
	ShadowMode bool
}
