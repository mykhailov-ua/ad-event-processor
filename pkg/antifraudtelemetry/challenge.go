package antifraudtelemetry

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"time"

	"github.com/google/uuid"
)

const (
	challengeVersion     = 1
	challengePayloadLen  = 38
	challengeMACLen      = 16
	challengeBinaryLen   = challengePayloadLen + challengeMACLen
	challengeTokenB64Len = 72
	challengeDefaultTTL  = 30
	challengeMinTTL      = 10
	challengeMaxTTL      = 120
	telemetryMACLen      = 16
	defaultPoWDifficulty = 2
	maxPoWDifficulty     = 4
)

var (
	ErrChallengeNoSecret = errors.New("antifraud challenge: missing secret")
	ErrChallengeMalformed = errors.New("antifraud challenge: malformed token")
	ErrChallengeExpired   = errors.New("antifraud challenge: expired")
	ErrChallengeCampaign  = errors.New("antifraud challenge: campaign mismatch")
	ErrPoWInvalid         = errors.New("antifraud challenge: pow invalid")
	ErrTelemetryMAC       = errors.New("antifraud challenge: telemetry mac invalid")
)

// MintChallenge returns a self-contained HMAC-signed PoW challenge for campaign_id.
func MintChallenge(secret []byte, campaignID uuid.UUID, nowUnix int64, ttlSec int, difficulty uint8) (string, error) {
	if len(secret) == 0 {
		return "", ErrChallengeNoSecret
	}
	ttl := clampChallengeTTL(ttlSec)
	if difficulty == 0 {
		difficulty = defaultPoWDifficulty
	}
	if difficulty > maxPoWDifficulty {
		difficulty = maxPoWDifficulty
	}
	var payload [challengePayloadLen]byte
	payload[0] = challengeVersion
	copy(payload[1:17], campaignID[:])
	var salt [16]byte
	if _, err := rand.Read(salt[:]); err != nil {
		h := sha256.Sum256(append(campaignID[:], byte(nowUnix)))
		copy(salt[:], h[:16])
	}
	copy(payload[17:33], salt[:])
	payload[33] = difficulty
	expires := uint32(nowUnix + int64(ttl))
	binary.BigEndian.PutUint32(payload[34:38], expires)
	mac := challengeMAC(secret, payload[:])
	var raw [challengeBinaryLen]byte
	copy(raw[:challengePayloadLen], payload[:])
	copy(raw[challengePayloadLen:], mac)
	return base64.RawURLEncoding.EncodeToString(raw[:]), nil
}

// MintChallengeWithSalt is for tests and deterministic verification.
func MintChallengeWithSalt(secret []byte, campaignID uuid.UUID, salt []byte, nowUnix int64, ttlSec int, difficulty uint8) (string, error) {
	if len(secret) == 0 {
		return "", ErrChallengeNoSecret
	}
	if len(salt) < 16 {
		return "", ErrChallengeMalformed
	}
	ttl := clampChallengeTTL(ttlSec)
	if difficulty == 0 {
		difficulty = defaultPoWDifficulty
	}
	var payload [challengePayloadLen]byte
	payload[0] = challengeVersion
	copy(payload[1:17], campaignID[:])
	copy(payload[17:33], salt[:16])
	payload[33] = difficulty
	expires := uint32(nowUnix + int64(ttl))
	binary.BigEndian.PutUint32(payload[34:38], expires)
	mac := challengeMAC(secret, payload[:])
	var raw [challengeBinaryLen]byte
	copy(raw[:challengePayloadLen], payload[:])
	copy(raw[challengePayloadLen:], mac)
	return base64.RawURLEncoding.EncodeToString(raw[:]), nil
}

type ChallengeInfo struct {
	CampaignID uuid.UUID
	Salt       [16]byte
	Difficulty uint8
	Expires    int64
}

func ParseChallengeToken(secret []byte, token string, campaignID uuid.UUID, nowUnix int64) (ChallengeInfo, error) {
	if len(secret) == 0 {
		return ChallengeInfo{}, ErrChallengeNoSecret
	}
	if len(token) != challengeTokenB64Len {
		return ChallengeInfo{}, ErrChallengeMalformed
	}
	var raw [challengeBinaryLen]byte
	n, err := base64.RawURLEncoding.Decode(raw[:], []byte(token))
	if err != nil || n != challengeBinaryLen {
		return ChallengeInfo{}, ErrChallengeMalformed
	}
	payload := raw[:challengePayloadLen]
	gotMAC := raw[challengePayloadLen:]
	if payload[0] != challengeVersion {
		return ChallengeInfo{}, ErrChallengeMalformed
	}
	expectMAC := challengeMAC(secret, payload)
	if subtle.ConstantTimeCompare(expectMAC, gotMAC) != 1 {
		return ChallengeInfo{}, ErrChallengeMalformed
	}
	var gotCampaign uuid.UUID
	copy(gotCampaign[:], payload[1:17])
	if gotCampaign != campaignID {
		return ChallengeInfo{}, ErrChallengeCampaign
	}
	expires := int64(binary.BigEndian.Uint32(payload[34:38]))
	if expires <= nowUnix {
		return ChallengeInfo{}, ErrChallengeExpired
	}
	var info ChallengeInfo
	info.CampaignID = gotCampaign
	copy(info.Salt[:], payload[17:33])
	info.Difficulty = payload[33]
	info.Expires = expires
	return info, nil
}

func VerifyPoW(salt []byte, nonce uint32, difficulty uint8) bool {
	if difficulty == 0 || difficulty > maxPoWDifficulty {
		return false
	}
	var buf [20]byte
	copy(buf[:16], salt[:16])
	binary.BigEndian.PutUint32(buf[16:20], nonce)
	sum := sha256.Sum256(buf[:20])
	for i := 0; i < int(difficulty); i++ {
		if sum[i] != 0 {
			return false
		}
	}
	return true
}

func PackTelemetryMACPayload(dwellMs uint32, pointerCV, rafCV uint16, runtimeLeak, automationLeak uint8) []byte {
	var scratch [12]byte
	binary.BigEndian.PutUint32(scratch[0:4], dwellMs)
	binary.BigEndian.PutUint16(scratch[4:6], pointerCV)
	binary.BigEndian.PutUint16(scratch[6:8], rafCV)
	scratch[8] = runtimeLeak
	scratch[9] = automationLeak
	return scratch[:10]
}

func DerivedTelemetryMACKey(challengeToken string, powNonce uint32) []byte {
	buf := make([]byte, len(challengeToken)+4)
	copy(buf, challengeToken)
	binary.BigEndian.PutUint32(buf[len(challengeToken):], powNonce)
	sum := sha256.Sum256(buf)
	return sum[:]
}

func ComputeTelemetryMACDerived(challengeToken string, powNonce uint32, dwellMs uint32, pointerCV, rafCV uint16, runtimeLeak, automationLeak uint8) [telemetryMACLen]byte {
	key := DerivedTelemetryMACKey(challengeToken, powNonce)
	payload := PackTelemetryMACPayload(dwellMs, pointerCV, rafCV, runtimeLeak, automationLeak)
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write(payload)
	sum := mac.Sum(nil)
	var out [telemetryMACLen]byte
	copy(out[:], sum[:telemetryMACLen])
	return out
}

func VerifyTelemetryMACDerived(challengeToken string, powNonce uint32, dwellMs uint32, pointerCV, rafCV uint16, runtimeLeak, automationLeak uint8, gotHex []byte) bool {
	if len(gotHex) != telemetryMACLen*2 {
		return false
	}
	var got [telemetryMACLen]byte
	for i := 0; i < telemetryMACLen; i++ {
		hi := fromHex(gotHex[i*2])
		lo := fromHex(gotHex[i*2+1])
		if hi < 0 || lo < 0 {
			return false
		}
		got[i] = byte(hi<<4 | lo)
	}
	expect := ComputeTelemetryMACDerived(challengeToken, powNonce, dwellMs, pointerCV, rafCV, runtimeLeak, automationLeak)
	return subtle.ConstantTimeCompare(expect[:], got[:]) == 1
}

// PackTelemetryMACInput is legacy secret-based MAC for tests only.
func PackTelemetryMACInput(challengeToken string, powNonce uint32, dwellMs uint32, pointerCV, rafCV uint16, runtimeLeak, automationLeak uint8) []byte {
	buf := make([]byte, 0, len(challengeToken)+16)
	buf = append(buf, challengeToken...)
	var scratch [12]byte
	binary.BigEndian.PutUint32(scratch[0:4], powNonce)
	binary.BigEndian.PutUint32(scratch[4:8], dwellMs)
	binary.BigEndian.PutUint16(scratch[8:10], pointerCV)
	binary.BigEndian.PutUint16(scratch[10:12], rafCV)
	buf = append(buf, scratch[:]...)
	buf = append(buf, runtimeLeak, automationLeak)
	return buf
}

func ComputeTelemetryMAC(secret []byte, input []byte) [telemetryMACLen]byte {
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write(input)
	sum := mac.Sum(nil)
	var out [telemetryMACLen]byte
	copy(out[:], sum[:telemetryMACLen])
	return out
}

func VerifyTelemetryMAC(secret []byte, input []byte, gotHex []byte) bool {
	if len(gotHex) != telemetryMACLen*2 {
		return false
	}
	var got [telemetryMACLen]byte
	for i := 0; i < telemetryMACLen; i++ {
		hi := fromHex(gotHex[i*2])
		lo := fromHex(gotHex[i*2+1])
		if hi < 0 || lo < 0 {
			return false
		}
		got[i] = byte(hi<<4 | lo)
	}
	expect := ComputeTelemetryMAC(secret, input)
	return subtle.ConstantTimeCompare(expect[:], got[:]) == 1
}

func challengeMAC(secret, payload []byte) []byte {
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write(payload)
	sum := mac.Sum(nil)
	return sum[:challengeMACLen]
}

func clampChallengeTTL(ttl int) int {
	if ttl < challengeMinTTL {
		return challengeDefaultTTL
	}
	if ttl > challengeMaxTTL {
		return challengeMaxTTL
	}
	return ttl
}

func fromHex(b byte) int {
	switch {
	case b >= '0' && b <= '9':
		return int(b - '0')
	case b >= 'a' && b <= 'f':
		return int(b - 'a' + 10)
	case b >= 'A' && b <= 'F':
		return int(b - 'A' + 10)
	default:
		return -1
	}
}
func ChallengeTTLDefault() int { return challengeDefaultTTL }

// DefaultPoWDifficulty exports default zero-byte prefix difficulty.
func DefaultPoWDifficulty() uint8 { return defaultPoWDifficulty }

// NowUnix is a test seam for time.Now().Unix().
var NowUnix = func() int64 { return time.Now().Unix() }
