"""
Faster-Whisper HTTP service for TranscriptFlow.

Dual-mode:
  1. HTTP  — POST /transcribe (multipart WAV) → JSON   [primary, used by Go layer-3]
  2. Redis — background thread pops from 'whisper:jobs', writes to 'whisper:results:<id>'

Response shape (both modes):
  {
    "text":     "full transcript text",
    "language": "en",
    "duration": 123.45,
    "segments": [{"start": 0.0, "end": 3.2, "text": "..."}, ...]
  }
The Go whisperResponse struct reads only {text, language, duration}; extra fields are ignored.
"""

import asyncio
import logging
import os
import tempfile
from contextlib import asynccontextmanager

from fastapi import FastAPI, File, HTTPException, UploadFile
from fastapi.responses import JSONResponse

from transcriber import Transcriber
from worker import start_redis_worker

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s %(levelname)s %(name)s: %(message)s",
)
logger = logging.getLogger(__name__)

ALLOWED_SUFFIXES = {".wav", ".mp3", ".m4a", ".ogg", ".flac", ".webm"}


@asynccontextmanager
async def lifespan(_: FastAPI):
    # Eagerly load the model before accepting traffic — avoids cold-start
    # latency on the first real request.
    Transcriber.get()
    # Start the Redis queue consumer in a background daemon thread.
    start_redis_worker()
    yield


app = FastAPI(
    title="TranscriptFlow Whisper Service",
    version="1.0.0",
    lifespan=lifespan,
)


@app.get("/health")
def health():
    return {"status": "ok", "model": "base.en"}


@app.post("/transcribe")
async def transcribe(file: UploadFile = File(...)):
    """
    Accept a 16 kHz mono PCM WAV file produced by the Go ffmpeg pipeline,
    run Faster-Whisper inference, and return structured JSON.

    The Go layer calls this via multipart/form-data with field name 'file'.
    """
    suffix = (
        os.path.splitext(file.filename or "audio.wav")[1].lower() or ".wav"
    )
    if suffix not in ALLOWED_SUFFIXES:
        raise HTTPException(status_code=400, detail=f"unsupported audio format: {suffix}")

    audio_bytes = await file.read()
    if len(audio_bytes) == 0:
        raise HTTPException(status_code=400, detail="empty file")

    # Write to a temp file; inference must operate on a path, not a buffer.
    with tempfile.NamedTemporaryFile(suffix=suffix, delete=False) as tmp:
        tmp.write(audio_bytes)
        tmp_path = tmp.name

    try:
        # Run CPU-bound inference in a thread pool so the event loop stays free.
        loop = asyncio.get_event_loop()
        result = await loop.run_in_executor(
            None, Transcriber.get().transcribe, tmp_path
        )
        logger.info(
            "HTTP transcribe complete: lang=%s duration=%.1fs segments=%d",
            result.language,
            result.duration,
            len(result.segments),
        )
        return JSONResponse(content=result.to_dict())
    except Exception as exc:
        logger.exception("Transcription failed: %s", exc)
        raise HTTPException(status_code=500, detail=f"transcription failed: {exc}")
    finally:
        try:
            os.unlink(tmp_path)
        except OSError:
            pass
