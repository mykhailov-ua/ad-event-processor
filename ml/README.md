# ML training / artifact bootstrap

Inference runs in Go: `cmd/fraud-scorer`, `internal/fraudscoring`.

```bash
# Dev / k8s PVC bootstrap (synthetic or copy testdata)
python3 ml/train.py bootstrap

# Refresh metadata.json from existing artifacts
python3 ml/train.py export
```

Docker image for k8s CronJob: `docker build -f deploy/ml/Dockerfile -t espx-ml-bootstrap:latest .`

Local optional deps: `pip install -r ml/requirements.txt`

Artifacts default to `var/fraudscore/artifacts/` (`FRAUD_ARTIFACT_DIR`).
