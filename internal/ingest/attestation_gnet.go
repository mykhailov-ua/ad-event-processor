package ingest

import (
	"time"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/ingest/parser"

	"github.com/google/uuid"
)

const attestationPayloadLen = 49

type attestationHMACKey struct {
	secret []byte
	ipad   [linkHMACBlockSize]byte
	opad   [linkHMACBlockSize]byte
}

func MintAttestationToken(secret []byte, campaignID uuid.UUID, clientIP string, ttlSec int32, nowUnix int64) (string, error) {
	return parser.MintAttestationToken(secret, campaignID, clientIP, ttlSec, nowUnix)
}

func (h *AdsPacketHandler) verifyAttestationCookie(cookieHeader []byte, campaignID uuid.UUID, clientIP string, nowUnix int64) bool {
	if h == nil || len(h.attestationKeys) == 0 {
		return false
	}
	keys := make([]parser.AttestationHMACKey, len(h.attestationKeys))
	for i := range h.attestationKeys {
		keys[i].Secret = h.attestationKeys[i].secret
		keys[i].Ipad = h.attestationKeys[i].ipad
		keys[i].Opad = h.attestationKeys[i].opad
	}
	return parser.VerifyAttestationToken(keys, h.attestationInnerScratch[:], cookieHeader, campaignID, clientIP, nowUnix)
}

func (h *AdsPacketHandler) ConfigureAttestation(secrets [][]byte) {
	if h == nil {
		return
	}
	h.attestationKeys = h.attestationKeys[:0]
	for _, secret := range secrets {
		if len(secret) == 0 {
			continue
		}
		var key attestationHMACKey
		key.secret = append([]byte(nil), secret...)
		linkInitHMACPads(key.secret, &key.ipad, &key.opad)
		h.attestationKeys = append(h.attestationKeys, key)
	}
}

func (h *AdsPacketHandler) campaignAttestationMode(campaignID uuid.UUID) domain.AttestationMode {
	if h == nil || h.registry == nil {
		return domain.AttestationModeOff
	}
	camp, ok := h.registry.GetCampaign(campaignID)
	if !ok || camp == nil || !camp.SafePageEnabled {
		return domain.AttestationModeOff
	}
	return domain.ResolveAttestationMode(camp.AttestationMode, camp.AttestationEnabled)
}

func (h *AdsPacketHandler) attestationRequired(campaignID uuid.UUID) bool {
	if h == nil || len(h.attestationKeys) == 0 {
		return false
	}
	return h.campaignAttestationMode(campaignID).RequiresProbe()
}

func buildAttestationSetCookie(token string, maxAge int32) []byte {
	if token == "" || maxAge <= 0 {
		return nil
	}
	prefix := []byte("Set-Cookie: Attestation-Token=")
	suffix := []byte("; Path=/click; Max-Age=")
	mid := []byte(token)
	tail := []byte("; HttpOnly; SameSite=Lax\r\n")
	var ageScratch [12]byte
	age := appendInt64(ageScratch[:0], int64(maxAge))
	out := make([]byte, 0, len(prefix)+len(mid)+len(suffix)+len(age)+len(tail))
	out = append(out, prefix...)
	out = append(out, mid...)
	out = append(out, suffix...)
	out = append(out, age...)
	out = append(out, tail...)
	return out
}

func (h *AdsPacketHandler) mintAttestationCookie(campaignID uuid.UUID, clientIP string) (string, int32) {
	if !h.attestationRequired(campaignID) {
		return "", 0
	}
	camp, _ := h.registry.GetCampaign(campaignID)
	ttl := campaignAttestationTTL(camp)
	token, err := MintAttestationToken(h.attestationKeys[0].secret, campaignID, clientIP, ttl, time.Now().Unix())
	if err != nil {
		return "", 0
	}
	return token, ttl
}

func campaignAttestationTTL(camp *domain.Campaign) int32 {
	if camp == nil || camp.AttestationTTLSec <= 0 {
		return 300
	}
	return parser.ClampAttestationTTL(camp.AttestationTTLSec)
}
