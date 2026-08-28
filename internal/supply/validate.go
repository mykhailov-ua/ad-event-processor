package supply

import (
	"fmt"
	"strings"
)

func NormalizeSellerType(v string) (string, error) {
	v = strings.ToUpper(strings.TrimSpace(v))
	switch v {
	case "PUBLISHER", "INTERMEDIARY", "BOTH":
		return v, nil
	default:
		return "", ErrInvalidSellerType
	}
}

func NormalizeRelationship(v string) (string, error) {
	v = strings.ToUpper(strings.TrimSpace(v))
	switch v {
	case "DIRECT", "RESELLER":
		return v, nil
	default:
		return "", ErrInvalidRelationship
	}
}

func ValidateChainNodes(nodes []ChainNode) error {
	if len(nodes) > MaxChainHops {
		return ErrChainTooLong
	}
	for i, n := range nodes {
		if strings.TrimSpace(n.ASI) == "" || strings.TrimSpace(n.SID) == "" {
			return fmt.Errorf("supply chain node %d: asi and sid are required", i)
		}
		if n.HP != 0 && n.HP != 1 {
			return fmt.Errorf("supply chain node %d: hp must be 0 or 1", i)
		}
	}
	return nil
}
