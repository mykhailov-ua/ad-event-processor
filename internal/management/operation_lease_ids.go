package management

import (
	"fmt"

	"github.com/google/uuid"
)

// RelayDeliveryOpID is the stable lease op_id for one regional outbox delivery.
func RelayDeliveryOpID(regionCode uint8, outboxEventID int64) uuid.UUID {
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(fmt.Sprintf("espx-relay-op:%d:%d", regionCode, outboxEventID)))
}

// ProxyBatchOpID returns the OpKeyPool op_id when set, otherwise a deterministic fallback.
func ProxyBatchOpID(regionCode uint8, nodeID string, seq uint64, opID uuid.UUID) uuid.UUID {
	if opID != uuid.Nil {
		return opID
	}
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(fmt.Sprintf("espx-proxy-op:%d:%s:%d", regionCode, nodeID, seq)))
}

// ProxyBatchOpIDFromBytes maps a 16-byte OpKey slot id to UUID.
func ProxyBatchOpIDFromBytes(opID [16]byte) uuid.UUID {
	var u uuid.UUID
	copy(u[:], opID[:])
	if u == uuid.Nil {
		return uuid.Nil
	}
	return u
}
