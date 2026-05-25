"""
Redis queue consumer for the Whisper service.

The Go backend (cmd/worker) can push jobs onto the 'whisper:jobs' Redis list
when it prefers fire-and-forget delivery over HTTP round-trips. Each job is a
JSON object:

    {
      "job_id":    "<uuid>",
      "audio_path": "/shared/audio_norm.wav"   # path visible to this container
    }

After transcription the result is written to 'whisper:results:<job_id>' with a
10-minute TTL so the Go worker can BLPOP or GET it.

The HTTP endpoint (app.py /transcribe) remains the primary path for Phase 5;
this worker provides the Redis-native path described in Phase 6.
"""

import json
import logging
import os
import threading
import time

import redis

from transcriber import Transcriber, TranscriptResult

logger = logging.getLogger(__name__)

REDIS_URL = os.getenv("REDIS_URL", "redis://localhost:6379")
JOBS_QUEUE = "whisper:jobs"
RESULTS_PREFIX = "whisper:results:"
RESULT_TTL_SECONDS = 600  # 10 minutes


def _process_job(rdb: redis.Redis, raw: str) -> None:
    job = json.loads(raw)
    job_id: str = job["job_id"]
    audio_path: str = job["audio_path"]
    language: str | None = job.get("language")

    logger.info("[worker] processing job_id=%s audio=%s", job_id, audio_path)

    result: TranscriptResult = Transcriber.get().transcribe(audio_path, language=language)

    result_key = RESULTS_PREFIX + job_id
    rdb.setex(result_key, RESULT_TTL_SECONDS, json.dumps(result.to_dict()))

    logger.info(
        "[worker] done job_id=%s lang=%s duration=%.1fs segments=%d",
        job_id,
        result.language,
        result.duration,
        len(result.segments),
    )


def _worker_loop() -> None:
    # Give the app a moment to finish startup before we start stealing CPU.
    time.sleep(2)

    rdb = redis.from_url(REDIS_URL, decode_responses=True, socket_keepalive=True)
    logger.info("[worker] connected to Redis, listening on '%s'", JOBS_QUEUE)

    while True:
        try:
            # BLPOP blocks until a job arrives (30 s timeout → loop to check alive).
            item = rdb.blpop(JOBS_QUEUE, timeout=30)
            if item is None:
                continue
            _, raw = item
            _process_job(rdb, raw)
        except redis.RedisError as exc:
            logger.error("[worker] Redis error: %s — retrying in 5 s", exc)
            time.sleep(5)
        except Exception:
            logger.exception("[worker] unexpected error — continuing")


def start_redis_worker() -> threading.Thread:
    """Start the Redis consumer in a daemon thread and return it."""
    t = threading.Thread(target=_worker_loop, daemon=True, name="redis-worker")
    t.start()
    return t
