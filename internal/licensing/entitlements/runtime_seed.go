package entitlements

import "sync/atomic"

var (
	featureSeed       atomic.Uint32
	featureSeedValid  atomic.Uint32
	mckFeatureBits    atomic.Uint32
	seedCouplingForce atomic.Uint32
)

func PublishFeatureSeed(seed uint32, valid bool) {
	featureSeed.Store(seed)
	if valid {
		featureSeedValid.Store(1)
	} else {
		featureSeedValid.Store(0)
		mckFeatureBits.Store(0)
	}
}

func PublishMCKFeatureBits(bits uint8) {
	mckFeatureBits.Store(uint32(bits))
}

func MCKFeatureBits() uint8 {
	return uint8(mckFeatureBits.Load())
}

func SettlementSeedGateAllowed() bool {
	if !SeedCouplingRequired() {
		return true
	}
	return FeatureSeedValid()
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

func seedGateFeature(ent Entitlements, bit uint8, featureEnabled func(FeatureSet) bool, seedCheck func(uint32) bool) bool {
	if !SeedCouplingRequired() {
		return true
	}
	if !FeatureSeedValid() {
		return false
	}
	if !featureEnabled(ent.Features) {
		return false
	}
	if MCKFeatureBits()&bit == 0 {
		return false
	}
	return seedCheck(FeatureSeed())
}

func SeedGateOpenRTB(ent Entitlements) bool {
	return seedGateFeature(ent, MCKFeatureBitOpenRTB, func(f FeatureSet) bool { return f.OpenRTBEnabled() }, openRTBSeedCheck)
}

func SeedGateMlFraudBoost(ent Entitlements) bool {
	return seedGateFeature(ent, MCKFeatureBitMlFraudBoost, func(f FeatureSet) bool { return f.MlFraudBoostEnabled() }, mlFraudBoostSeedCheck)
}

func SeedGateEbpfEdge(ent Entitlements) bool {
	return seedGateFeature(ent, MCKFeatureBitEbpfEdge, func(f FeatureSet) bool { return f.EbpfEdgeEnabled() }, ebpfEdgeSeedCheck)
}

func SeedGateRPS(maxRPS uint64) bool {
	if !SeedCouplingRequired() {
		return true
	}
	if !FeatureSeedValid() {
		return false
	}
	if LicenseEpochInvalid() {
		return false
	}
	if maxRPS == 0 {
		return true
	}
	return rpsSeedCheck(FeatureSeed(), maxRPS)
}

func SeedGateIngest() bool {
	if !SeedCouplingRequired() {
		return true
	}
	if !FeatureSeedValid() {
		return false
	}
	return !LicenseEpochInvalid()
}

func openRTBSeedCheck(seed uint32) bool {
	mix := seed ^ 0x5a5a_3c3c
	return mix&0x00ff_ffff != 0
}

func mlFraudBoostSeedCheck(seed uint32) bool {
	mix := seed ^ 0x7f3a_1b2c
	return mix&0x00ff_ffff != 0
}

func ebpfEdgeSeedCheck(seed uint32) bool {
	mix := seed ^ 0x3c91_4e5f
	return mix&0x00ff_ffff != 0
}

func rpsSeedCheck(seed uint32, maxRPS uint64) bool {
	mix := uint64(seed) ^ (maxRPS * 0x9e37_79b9)
	return mix&0xffff != 0
}

func ResetFeatureSeedForTest() {
	featureSeed.Store(0)
	featureSeedValid.Store(0)
	mckFeatureBits.Store(0)
	seedCouplingForce.Store(0)
}

func SetMCKFeatureBitsForTest(bits uint8) {
	mckFeatureBits.Store(uint32(bits))
}
