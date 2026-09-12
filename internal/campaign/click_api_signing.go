package campaign

import (
	"time"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/track"
)

type ProgrammaticClickLinkSigning struct {
	Enabled            bool
	TTLSec             int32
	AttestationMode    string
	AttestationEnabled bool
	Secret             []byte
}

func MaybeSignProgrammaticClickURL(clickURL, clickID string, signing ProgrammaticClickLinkSigning) string {
	if !signing.Enabled || len(signing.Secret) == 0 || clickURL == "" || clickID == "" {
		return clickURL
	}
	camp := &domain.Campaign{
		LinkSigningTTLSec:  signing.TTLSec,
		AttestationMode:    domain.ParseAttestationMode(signing.AttestationMode),
		AttestationEnabled: signing.AttestationEnabled,
	}
	expires := track.LinkSigningExpires(time.Now(), track.EffectiveLinkSigningTTLSec(camp))
	return string(track.AppendLinkSignature([]byte(clickURL), signing.Secret, []byte(clickID), expires))
}
