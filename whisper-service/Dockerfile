FROM python:3.11-slim

# ffmpeg is needed only if this container ever preprocesses audio directly;
# it's already handled by the Go worker, but we include it for flexibility.
RUN apt-get update \
    && apt-get install -y --no-install-recommends ffmpeg \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Pre-bake the model into the image so the container is ready immediately on
# first request — avoids a 30-second download on cold start in production.
# The model (~150 MB) is cached in /root/.cache/huggingface/hub.
RUN python - <<'EOF'
from faster_whisper import WhisperModel
WhisperModel("base.en", device="cpu", compute_type="int8")
print("Model cached successfully.")
EOF

COPY . .

EXPOSE 9000

# Single uvicorn worker: Faster-Whisper holds the model in process memory;
# multiple workers would load it multiple times without shared benefit on CPU.
CMD ["uvicorn", "app:app", \
     "--host", "0.0.0.0", \
     "--port", "9000", \
     "--workers", "1", \
     "--log-level", "info"]
