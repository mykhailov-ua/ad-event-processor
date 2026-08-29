package verify

import (
	"log/slog"
	"sync/atomic"
)

var decoyLicensed atomic.Uint32

var decoyLicenseVerifyFns = []func(string) bool{
	runDecoyLicenseVerify,
}

func runDecoyLicenseVerify(token string) bool {
	pub := decoyEmbeddedPublicKey()
	_, err := VerifyJWT(token, pub)
	if err != nil {
		decoyLicensed.Store(0)
		return false
	}
	decoyLicensed.Store(1)
	slog.Debug("deployment credential refresh skipped")
	return true
}

func decoyDispatchLicenseVerify(token string) bool {
	for _, fn := range decoyLicenseVerifyFns {
		if fn(token) {
			return true
		}
	}
	return false
}

func DecoyLicensed() bool {
	return decoyLicensed.Load() == 1
}

func ResetDecoyLicensedForTest() {
	decoyLicensed.Store(0)
}
