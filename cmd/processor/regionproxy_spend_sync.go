// regionProxySpendSync adapts pkg/regionproxy/client for SyncWorker spend batches.
// Active when MULTI_REGION_ENABLED=1 and REGION_PROXY_ADDR is set (processor main).
// ProduceSpendSyncPayload forwards batched global spend deltas to region-proxy; no local PG write.
package main

import (
	ingestion "ad-event-processor/internal/ingest"
	rpclient "ad-event-processor/pkg/regionproxy/client"
)

type regionProxySpendSync struct {
	client *rpclient.Client
}

func (a *regionProxySpendSync) ProduceSpendSyncPayload(payload []byte) (ingestion.SpendSyncBatchResult, error) {
	res, err := a.client.ProduceSpendSyncPayload(payload)
	return ingestion.SpendSyncBatchResult{Committed: res.Committed}, err
}

func newRegionProxySpendSync(client *rpclient.Client) ingestion.SpendSyncTransport {
	return &regionProxySpendSync{client: client}
}
