"""Shared fraud ML contracts mirrored from internal/fraud/*.go.

Role:
- feature_spec: 16-dim vector layout from ml_features_1m aggregates.
- policy_config: FRAUD_POLICY_* env and metadata.json policy section.
- scoring_policy: post-ML heuristics (proxy boost, structural bypass, FP guard).
- fixture_catalog: canonical row fixtures for vector and ml-validate tests.

Forbidden:
- Do not change FEATURE_NAMES order without updating internal/fraud/feature_spec.go
  and regenerating fixtures (model/train/fixture_generator.py).

Verify:
  pytest model/tests/test_feature_spec.py model/tests/test_scoring_policy_parity.py -q
  go test ./internal/fraud/ -short -run TestFeatureSpec -count=1
"""
