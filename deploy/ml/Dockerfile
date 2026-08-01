FROM python:3.12-slim
WORKDIR /app
COPY fraudtrain/requirements.txt fraudtrain/requirements.txt
RUN pip install --no-cache-dir -r fraudtrain/requirements.txt
COPY fraudtrain/ fraudtrain/
ENV FRAUD_ARTIFACT_DIR=/var/fraudscore/artifacts
ENTRYPOINT ["python3", "fraudtrain/artifact_bootstrap.py"]
CMD ["fit-validate"]
