FROM python:3.12-slim
WORKDIR /app
COPY model/requirements.txt model/requirements.txt
RUN pip install --no-cache-dir -r model/requirements.txt
COPY model/ model/
ENV FRAUD_ARTIFACT_DIR=/var/fraudscore/artifacts
ENTRYPOINT ["python3", "model/artifact_bootstrap.py"]
