package stream

import "time"

func (c *UDPControl) SetSyncIntervalForTest(d time.Duration) {
	if c != nil {
		c.syncInterval = d
	}
}

func (c *UDPControl) MarkFreshForTest() {
	if c != nil {
		c.markFresh()
	}
}

func (c *UDPControl) SetLastPacketMonoForTest(v int64) {
	if c != nil {
		c.lastPacketMono.Store(v)
	}
}

func (c *UDPControl) CheckStaleForTest() {
	if c != nil {
		c.checkStale()
	}
}

func (c *UDPControl) SetLastPublisherEpochForTest(v int64) {
	if c != nil {
		c.lastPublisherEpoch.Store(v)
	}
}
