package payment

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"testing"
	"time"
)

func BenchmarkVerifyStripeSignature(b *testing.B) {
	secret := "stripe_wh_secret"
	timestamp := "1672531199"
	payload := []byte(`{"id":"evt_stripe_test_999","type":"payment_intent.succeeded"}`)

	signedPayload := []byte(timestamp + "." + string(payload))
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(signedPayload)
	expectedMAC := mac.Sum(nil)
	sig := hex.EncodeToString(expectedMAC)
	sigHeader := fmt.Sprintf("t=%s,v1=%s", timestamp, sig)
	for b.Loop() {
		_ = verifyStripeSignature(payload, sigHeader, secret, time.Unix(1672531199, 0))
	}
}
