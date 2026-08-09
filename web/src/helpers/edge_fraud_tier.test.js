import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
import {
  fraudTierFromScore,
  FRAUD_TIER_PASS_MAX,
  FRAUD_TIER_SUSPECT_MAX,
  FRAUD_TIER_IVT_MAX,
} from './edge_fraud_tier.js';

describe('fraudTierFromScore', () => {
  it('matches edge-fraud-tier.lua bands', () => {
    assert.equal(fraudTierFromScore(10).tier, 'pass');
    assert.equal(fraudTierFromScore(FRAUD_TIER_PASS_MAX).tier, 'pass');
    assert.equal(fraudTierFromScore(FRAUD_TIER_PASS_MAX + 1).tier, 'suspect');
    assert.equal(fraudTierFromScore(FRAUD_TIER_SUSPECT_MAX).tier, 'suspect');
    assert.equal(fraudTierFromScore(FRAUD_TIER_SUSPECT_MAX + 1).tier, 'ivt');
    assert.equal(fraudTierFromScore(FRAUD_TIER_IVT_MAX).tier, 'ivt');
    assert.equal(fraudTierFromScore(FRAUD_TIER_IVT_MAX + 1).tier, 'block');
  });
});
