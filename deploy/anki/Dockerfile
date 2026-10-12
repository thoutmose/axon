# syntax=docker/dockerfile:1
# Axon's Anki bridge (EP-42): the official Anki library behind a tiny HTTP API.
FROM python:3.12-slim@sha256:a6e34c598f2467ed0e9a8d349809fcd8b5c603269512df273a0bb1784edc11b1
ENV PYTHONDONTWRITEBYTECODE=1 PYTHONUNBUFFERED=1 PIP_NO_CACHE_DIR=1 PIP_ROOT_USER_ACTION=ignore
RUN pip install "anki==26.9.3" && useradd --system --uid 10001 --home /data anki && install -d -o anki -g anki /data
COPY bridge.py /app/bridge.py
USER anki
VOLUME /data
EXPOSE 8090
HEALTHCHECK --interval=30s --timeout=3s CMD python -c "import urllib.request; urllib.request.urlopen('http://127.0.0.1:8090/healthz')"
CMD ["python", "/app/bridge.py"]
