package dedupkey

import (
	"fmt"

	"github.com/google/uuid"
)

type Scope struct {
	RegionID    uuid.UUID
	SourceID    uuid.UUID
	SourceEpoch uint32
	SeqStart    int64
	SeqEnd      int64
}

func RegionUUID(regionCode uint8) uuid.UUID {
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(fmt.Sprintf("espx-region:%d", regionCode)))
}

func SyncWorkerSourceID(shardID int16, campaignID uuid.UUID) uuid.UUID {
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(fmt.Sprintf("espx-sync:%d:%s", shardID, campaignID)))
}

func RelaySourceID(regionCode uint8) uuid.UUID {
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(fmt.Sprintf("espx-relay:%d", regionCode)))
}

func BrokerSourceID(topic string, partition uint16, group string) uuid.UUID {
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(fmt.Sprintf("espx-broker:%s:%d:%s", topic, partition, group)))
}

func InflightSeq(txID string) int64 {
	if txID == "" {
		return 0
	}
	u, err := uuid.Parse(txID)
	if err == nil {
		return int64(u.ID())
	}
	h := FactorU([]byte("txid:" + txID))
	return int64(h.ID())
}
