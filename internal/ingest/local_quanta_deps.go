package ingest

// LocalQuantaDepsWithStream wires concrete stream/idempotency method values for Tier B
// local-quanta full-skip (avoids per-request interface boxing on TryClaim and Enqueue).
func LocalQuantaDepsWithStream(ledger *LocalQuantaLedger, stream *LocalQuantaStreamPublisher) LocalQuantaDeps {
	deps := LocalQuantaDeps{}
	if ledger != nil {
		deps.Ledger = ledger
	}
	if stream == nil {
		return deps
	}
	idem := stream.IdemCache()
	deps.Stream = stream
	deps.Idem = idem
	deps.StreamHot = stream
	deps.IdemHot = idem
	deps.ClickTryClaim = idem.TryClaim
	deps.ClickRelease = idem.Release
	deps.StreamEnqueue = stream.Enqueue
	return deps
}
