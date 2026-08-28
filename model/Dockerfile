FROM python:3.12-slim
WORKDIR /app
COPY requirements.txt requirements.txt
RUN pip install --no-cache-dir -r requirements.txt
COPY . .
ENV FRAUD_ARTIFACT_DIR=/var/fraudscore/artifacts
ENV PYTHONPATH=/app
ENTRYPOINT ["python3", "-m", "train.artifact_bootstrap"]
