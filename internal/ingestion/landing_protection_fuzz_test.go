package ingestion

import (
	"testing"
)

func FuzzJA3Parse(f *testing.F) {
	f.Add("ja3:771,4865-4866")
	f.Add("ja4:abc,def")
	f.Add("t13d1516h2=chrome,firefox")
	f.Fuzz(func(t *testing.T, line string) {
		ja3, ja4 := parseTLSFingerprintFeed([]byte(line))
		table := NewTLSFingerprintTable()
		table.Publish(buildTLSFingerprintSnapshot(ja3, ja4, nil, nil, 1))
		table.MatchJA3([]byte(line))
		table.MatchJA4([]byte(line))
		parseJA4BrowserCorpus([]byte(line))
		ja4BrowserCorpusMismatch("Mozilla/5.0 Chrome/120.0.0.0", []byte("t13d1516h2_aaaaaaaaaaaa_bbbbbbbbbbbb"))
	})
}

func FuzzLinkSignerVerify(f *testing.F) {
	f.Add("secret", "click", int64(1_700_000_000), "abcdef0123456789abcdef0123456789")
	f.Fuzz(func(t *testing.T, secret, clickID string, expires int64, sig string) {
		VerifyLinkSignature([]byte(secret), []byte(clickID), []byte(sig), expires, expires)
	})
}

func FuzzSafePageVerifyParse(f *testing.F) {
	f.Add(`{"campaign_id":"550e8400-e29b-41d4-a716-446655440000","events":[{"t":"mousemove","ts":1,"x":1,"y":2}],"fingerprint":{"ua":"x","lang":"en","languages":["en"],"canvas_hash":"abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789","audio_hash":"fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210","notification_permission":"denied","notification_query":"denied"}}`)
	f.Fuzz(func(t *testing.T, body string) {
		req, ok := parseSafePageVerifyRequest([]byte(body))
		if !ok {
			return
		}
		evaluateSafePageAttestation(safePageAttestationInput{
			fingerprint: req.Fingerprint,
			events:      req.Events,
		})
		checkBezierBot(req.Events)
	})
}
