package management

import (
	"fmt"

	"github.com/google/uuid"
)

func RelayDeliveryOpID(regionCode uint8, outboxEventID int64) uuid.UUID {
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(fmt.Sprintf("espx-relay-op:%d:%d", regionCode, outboxEventID)))
}

func ProxyBatchOpID(regionCode uint8, nodeID string, seq uint64, opID uuid.UUID) uuid.UUID {
	if opID != uuid.Nil {
		return opID
	}
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(fmt.Sprintf("espx-proxy-op:%d:%s:%d", regionCode, nodeID, seq)))
}

func ProxyBatchOpIDFromBytes(opID [16]byte) uuid.UUID {
	var u uuid.UUID
	copy(u[:], opID[:])
	if u == uuid.Nil {
		return uuid.Nil
	}
	return u
}
