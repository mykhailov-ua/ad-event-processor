package governance

type OutboxWorker struct {
	host Host
}

func NewOutboxWorker(host Host) *OutboxWorker {
	return &OutboxWorker{host: host}
}
