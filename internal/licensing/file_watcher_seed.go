package licensing

import (
	"crypto/ed25519"

	"ad-event-processor/internal/config"
)

func publishFeatureSeedAfterVerify(path string, pubKey ed25519.PublicKey, verifyErr error) {
	SetSeedCouplingRequired(config.LicenseSeedCouplingEnabled())
	seed, mckBits, seedValid := featureSeedFromRecheck(path, pubKey, HostFingerprint(), verifyErr)
	PublishFeatureSeed(seed, seedValid)
	if seedValid {
		PublishMCKFeatureBits(mckBits)
	}
}
