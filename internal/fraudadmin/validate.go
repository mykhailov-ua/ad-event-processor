package fraudadmin

import (
	"unicode"
)

func ValidateThresholds(pass, suspect, ivt, block uint8) error {
	if pass > 100 || suspect > 100 || ivt > 100 || block > 100 {
		return ValidationError("fraud thresholds must be between 0 and 100")
	}
	if pass > suspect || suspect > ivt || ivt > block {
		return ValidationError("fraud thresholds must be ordered: pass <= suspect <= ivt <= block")
	}
	return nil
}

func ValidMLIPHashHex(s string) bool {
	if len(s) != 32 {
		return false
	}
	for _, r := range s {
		if !unicode.IsDigit(r) && (r < 'a' || r > 'f') {
			return false
		}
	}
	return true
}

func ValidateMLIPHash(s string) error {
	if !ValidMLIPHashHex(s) {
		return ValidationError("ip_hash must be 32 hex characters")
	}
	return nil
}

const (
	ManualLabelsDefaultLimit = 50
	ManualLabelsMaxLimit     = 100
	ManualLabelsBulkMax      = 500
	ExplainMaxHours          = 168
	ThreatBatchMax           = 500
)

const DecisionDisclaimer = "Decision as of last scorer window; replay uses stored features and shadow ML score."
