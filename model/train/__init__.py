"""Training bootstrap, datasets, policy calibration, and fixture generation.

Role:
- artifact_bootstrap: LightGBM + iforest artifacts, metadata.json, policy calibration.
- labeled_dataset: time-ordered train/val split from features_export output.
- policy_calibrate: grid search over heuristic knobs under FPR caps.
- traffic_simulator: synthetic archetypes for bootstrap when CH labels absent.
- fixture_generator: write features_*.json for cmd/ml-validate parity.

Artifact dir: FRAUD_ARTIFACT_DIR or var/fraudscore/artifacts/

Verify:
  python3 model/train/artifact_bootstrap.py bootstrap-validate
  pytest model/tests/test_bootstrap_contract.py model/tests/test_labeled_dataset.py -q
"""
