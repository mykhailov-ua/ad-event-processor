package licensing

import "sync/atomic"

var (
	featureSeed       atomic.Uint32
	featureSeedValid  atomic.Uint32
	seedCouplingForce atomic.Uint32
)

func PublishFeatureSeed(seed uint32, valid bool) {
	featureSeed.Store(seed)
	if valid {
		featureSeedValid.Store(1)
	} else {
		featureSeedValid.Store(0)
	}
}

func FeatureSeed() uint32 {
	return featureSeed.Load()
}

func FeatureSeedValid() bool {
	return featureSeedValid.Load() == 1
}

func SetSeedCouplingRequired(required bool) {
	if required {
		seedCouplingForce.Store(1)
	} else {
		seedCouplingForce.Store(0)
	}
}

func SeedCouplingRequired() bool {
	return seedCouplingForce.Load() == 1
}

func SeedGateOpenRTB(ent Entitlements) bool {
	if !SeedCouplingRequired() {
		return true
	}
	if !FeatureSeedValid() {
		return false
	}
	if !ent.Features.OpenRTBEnabled() {
		return false
	}
	seed := FeatureSeed()
	return openRTBSeedCheck(seed)
}

func SeedGateRPS(maxRPS uint64) bool {
	if !SeedCouplingRequired() || maxRPS == 0 {
		return true
	}
	if !FeatureSeedValid() {
		return false
	}
	return rpsSeedCheck(FeatureSeed(), maxRPS)
}

func openRTBSeedCheck(seed uint32) bool {
	mix := seed ^ 0x5a5a_3c3c
	return mix&0x00ff_ffff != 0
}

func rpsSeedCheck(seed uint32, maxRPS uint64) bool {
	mix := uint64(seed) ^ (maxRPS * 0x9e37_79b9)
	return mix&0xffff != 0
}

func ResetFeatureSeedForTest() {
	featureSeed.Store(0)
	featureSeedValid.Store(0)
	seedCouplingForce.Store(0)
}
