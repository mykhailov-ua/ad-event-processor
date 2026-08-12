package main

import (
	"github.com/bidshard/ad-event-processor/internal/ingestion"
	rpclient "github.com/bidshard/ad-event-processor/pkg/regionproxy/client"
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
