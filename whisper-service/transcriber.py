"""
Singleton wrapper around Faster-Whisper.

The model is loaded once on first access and reused for all subsequent calls.
faster_whisper.WhisperModel is thread-safe for concurrent inference, so the
singleton is safe even when the Redis worker and HTTP handlers call it together.
"""

import logging
import os
import threading
from dataclasses import dataclass
from typing import Optional

from faster_whisper import WhisperModel

logger = logging.getLogger(__name__)

MODEL_SIZE = os.getenv("WHISPER_MODEL", "base.en")
DEVICE = os.getenv("WHISPER_DEVICE", "cpu")
COMPUTE_TYPE = os.getenv("WHISPER_COMPUTE_TYPE", "int8")
BEAM_SIZE = int(os.getenv("WHISPER_BEAM_SIZE", "5"))


@dataclass
class Segment:
    start: float
    end: float
    text: str


@dataclass
class TranscriptResult:
    text: str
    language: str
    duration: float
    segments: list[Segment]

    def to_dict(self) -> dict:
        return {
            "text": self.text,
            "language": self.language,
            "duration": round(self.duration, 3),
            "segments": [
                {"start": round(s.start, 3), "end": round(s.end, 3), "text": s.text}
                for s in self.segments
            ],
        }


class Transcriber:
    """Thread-safe Faster-Whisper singleton."""

    _instance: Optional["Transcriber"] = None
    _lock = threading.Lock()

    def __init__(self) -> None:
        logger.info(
            "Loading Faster-Whisper model=%s device=%s compute_type=%s",
            MODEL_SIZE,
            DEVICE,
            COMPUTE_TYPE,
        )
        self._model = WhisperModel(MODEL_SIZE, device=DEVICE, compute_type=COMPUTE_TYPE)
        logger.info("Model ready.")

    @classmethod
    def get(cls) -> "Transcriber":
        if cls._instance is None:
            with cls._lock:
                if cls._instance is None:
                    cls._instance = cls()
        return cls._instance

    def transcribe(self, audio_path: str, language: Optional[str] = None) -> TranscriptResult:
        """
        Run inference on a WAV file.
        Caller must supply a 16 kHz mono PCM WAV — the format normalised by ffmpeg
        in the Go layer-3 pipeline.
        """
        kwargs: dict = {"beam_size": BEAM_SIZE, "vad_filter": True}
        if language:
            kwargs["language"] = language

        segments_gen, info = self._model.transcribe(audio_path, **kwargs)

        segments: list[Segment] = []
        text_parts: list[str] = []

        for seg in segments_gen:
            clean = seg.text.strip()
            if not clean:
                continue
            segments.append(Segment(start=seg.start, end=seg.end, text=clean))
            text_parts.append(clean)

        return TranscriptResult(
            text=" ".join(text_parts),
            language=info.language,
            duration=info.duration,
            segments=segments,
        )
