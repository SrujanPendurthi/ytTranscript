package transcript

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type whisperLayer struct {
	url    string
	client *http.Client
}

func newWhisperLayer(whisperURL string) *whisperLayer {
	return &whisperLayer{
		url: whisperURL,
		client: &http.Client{
			// Whisper inference can be slow on CPU — allow 10 minutes.
			Timeout: 10 * time.Minute,
		},
	}
}

// Transcribe runs the full fallback pipeline:
//  1. Verify yt-dlp and ffmpeg are on PATH.
//  2. yt-dlp downloads the best audio stream to a temp WAV.
//  3. ffmpeg re-encodes to 16 kHz mono PCM (the format Faster-Whisper expects).
//  4. The normalised file is POSTed to the Faster-Whisper service.
func (w *whisperLayer) Transcribe(ctx context.Context, youtubeURL string) (*Result, error) {
	if _, err := exec.LookPath("yt-dlp"); err != nil {
		return nil, fmt.Errorf("yt-dlp not found on PATH: %w", err)
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return nil, fmt.Errorf("ffmpeg not found on PATH: %w", err)
	}

	tmpDir, err := os.MkdirTemp("", "tf-whisper-*")
	if err != nil {
		return nil, fmt.Errorf("mktemp: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Step 1: download audio
	rawAudio, err := downloadAudio(ctx, youtubeURL, tmpDir)
	if err != nil {
		return nil, fmt.Errorf("yt-dlp: %w", err)
	}

	// Step 2: normalise to 16 kHz mono PCM WAV
	normAudio := filepath.Join(tmpDir, "audio_norm.wav")
	if err := normalizeAudio(ctx, rawAudio, normAudio); err != nil {
		return nil, fmt.Errorf("ffmpeg normalize: %w", err)
	}

	// Step 3: send to Faster-Whisper service
	result, err := w.postToWhisper(ctx, normAudio)
	if err != nil {
		return nil, fmt.Errorf("whisper service: %w", err)
	}
	return result, nil
}

// downloadAudio invokes yt-dlp to extract the best audio and convert to WAV.
// Returns the path of the produced WAV file.
func downloadAudio(ctx context.Context, youtubeURL, tmpDir string) (string, error) {
	outputTemplate := filepath.Join(tmpDir, "audio.%(ext)s")
	cmd := exec.CommandContext(ctx,
		"yt-dlp",
		"--extract-audio",
		"--audio-format", "wav",
		"--audio-quality", "0",
		"--no-playlist",
		"--quiet",
		"--no-warnings",
		"-o", outputTemplate,
		youtubeURL,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("%w\nyt-dlp output: %s", err, out)
	}

	// Find the produced WAV (yt-dlp substitutes %(ext)s → wav).
	matches, err := filepath.Glob(filepath.Join(tmpDir, "audio.*"))
	if err != nil || len(matches) == 0 {
		return "", fmt.Errorf("yt-dlp produced no output file")
	}
	return matches[0], nil
}

// normalizeAudio re-encodes the input to 16 kHz mono 16-bit PCM WAV.
func normalizeAudio(ctx context.Context, input, output string) error {
	cmd := exec.CommandContext(ctx,
		"ffmpeg",
		"-i", input,
		"-ar", "16000",      // 16 kHz sample rate
		"-ac", "1",          // mono channel
		"-acodec", "pcm_s16le", // 16-bit little-endian PCM
		"-y",                // overwrite output without asking
		"-loglevel", "error",
		output,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%w\nffmpeg output: %s", err, out)
	}
	return nil
}

type whisperSegment struct {
	Start float64 `json:"start"`
	End   float64 `json:"end"`
	Text  string  `json:"text"`
}

type whisperResponse struct {
	Text     string           `json:"text"`
	Language string           `json:"language"`
	Duration float64          `json:"duration"`
	Segments []whisperSegment `json:"segments"`
}

// postToWhisper sends the normalised WAV to the Faster-Whisper HTTP service
// and returns the transcript result.
// The service (Phase 6) must accept multipart/form-data with a "file" field
// and respond with {"text":"…","language":"…","duration":…}.
func (w *whisperLayer) postToWhisper(ctx context.Context, audioPath string) (*Result, error) {
	f, err := os.Open(audioPath)
	if err != nil {
		return nil, fmt.Errorf("open audio: %w", err)
	}
	defer f.Close()

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)

	fw, err := mw.CreateFormFile("file", filepath.Base(audioPath))
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(fw, f); err != nil {
		return nil, err
	}
	mw.Close()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.url+"/transcribe", &body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())

	resp, err := w.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http post: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<10))
		return nil, fmt.Errorf("whisper returned %d: %s", resp.StatusCode, raw)
	}

	var wr whisperResponse
	if err := json.NewDecoder(resp.Body).Decode(&wr); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	segs := make([]Segment, len(wr.Segments))
	for i, s := range wr.Segments {
		segs[i] = Segment{Start: s.Start, End: s.End, Text: s.Text}
	}
	return &Result{
		Text:        wr.Text,
		Language:    wr.Language,
		Duration:    wr.Duration,
		WhisperUsed: true,
		Segments:    segs,
		Source:      "whisper",
	}, nil
}
