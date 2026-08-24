package ingestion

import (
	"testing"

	"ad-event-processor/internal/domain"

	"github.com/stretchr/testify/assert"
)

func TestEffectiveLinkSigningTTLSec_attestationCaps(t *testing.T) {
	camp := &domain.Campaign{
		LinkSigningTTLSec: 900,
		AttestationMode:   domain.AttestationModeLight,
	}
	assert.Equal(t, int32(300), EffectiveLinkSigningTTLSec(camp))

	camp.AttestationMode = domain.AttestationModeOff
	assert.Equal(t, int32(900), EffectiveLinkSigningTTLSec(camp))

	camp.AttestationMode = domain.AttestationModeStrict
	camp.LinkSigningTTLSec = 120
	assert.Equal(t, int32(120), EffectiveLinkSigningTTLSec(camp))
}
