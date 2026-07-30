FROM python:3.12-slim
WORKDIR /app
COPY ml/requirements.txt ml/requirements.txt
RUN pip install --no-cache-dir -r ml/requirements.txt
COPY ml/train.py ml/train.py
COPY internal/fraudscoring/testdata/model.txt internal/fraudscoring/testdata/model.txt
ENV FRAUD_ARTIFACT_DIR=/var/fraudscore/artifacts
ENTRYPOINT ["python3", "ml/train.py"]
CMD ["bootstrap"]
