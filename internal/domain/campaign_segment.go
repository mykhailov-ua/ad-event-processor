package domain

import (
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func applyCampaignSegmentFields(
	camp *Campaign,
	retarget, include, exclude pgtype.UUID,
	ttlHours int32,
) {
	if camp == nil {
		return
	}
	if retarget.Valid {
		camp.RetargetSegmentID = uuid.UUID(retarget.Bytes)
	}
	camp.SegmentTTLHours = ttlHours
	if include.Valid {
		camp.SegmentIncludeID = uuid.UUID(include.Bytes)
	}
	if exclude.Valid {
		camp.SegmentExcludeID = uuid.UUID(exclude.Bytes)
	}
}

func parseConnTypePolicy(s string) ConnTypePolicy {
	return ConnTypePolicyFromString(s)
}

func ConnTypePolicyFromString(s string) ConnTypePolicy {
	switch ConnTypePolicy(s) {
	case ConnTypeMobileOnly, ConnTypeResidentialOnly, ConnTypeBlockVPNHosting:
		return ConnTypePolicy(s)
	default:
		return ConnTypeBlockVPNHosting
	}
}

func applyCampaignGMAFields(c *Campaign, tlsBlock bool, connPolicy string, linkSign bool, linkTTL int32) {
	if c == nil {
		return
	}
	c.TLSFingerprintBlockEnabled = tlsBlock
	c.ConnTypePolicy = parseConnTypePolicy(connPolicy)
	c.LinkSigningEnabled = linkSign
	c.LinkSigningTTLSec = linkTTL
}
