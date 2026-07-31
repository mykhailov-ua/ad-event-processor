FROM python:3.12-slim
WORKDIR /app
COPY fraud_modeling/requirements.txt fraud_modeling/requirements.txt
RUN pip install --no-cache-dir -r fraud_modeling/requirements.txt
COPY fraud_modeling/ fraud_modeling/
ENV FRAUD_ARTIFACT_DIR=/var/fraudscore/artifacts
ENTRYPOINT ["python3", "fraud_modeling/artifact_bootstrap.py"]
CMD ["fit-validate"]
