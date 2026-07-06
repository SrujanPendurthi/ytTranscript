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
	"sort"
	"time"
)

// openAITranscribeURL is the OpenAI audio transcription endpoint.
const openAITranscribeURL = "https://api.openai.com/v1/audio/transcriptions"

// openAIMaxUploadBytes is OpenAI's 25 MB per-file limit. We split audio into
// time-based chunks and keep each comfortably under this ceiling.
const openAIMaxUploadBytes = 24 << 20 // 24 MB

// chunkSeconds is the per-chunk duration used when an audio file exceeds the
// upload limit. At the 64 kbps mono MP3 we encode, 20 min ≈ 9.6 MB — well under 25 MB.
const chunkSeconds = 1200

// whisperLayer is the audio-transcription fallback. It downloads and normalises
// the video's audio, then transcribes it via the OpenAI Whisper API.
type whisperLayer struct {
	apiKey string
	client *http.Client
}

func newWhisperLayer(apiKey string) *whisperLayer {
	return &whisperLayer{
		apiKey: apiKey,
		client: &http.Client{
			// Whisper inference plus upload can be slow — allow 10 minutes.
			Timeout: 10 * time.Minute,
		},
	}
}

// Transcribe runs the full fallback pipeline:
//  1. Verify yt-dlp and ffmpeg are on PATH.
//  2. yt-dlp downloads the best audio stream.
//  3. ffmpeg re-encodes to a compact 16 kHz mono MP3 (small enough to upload).
//  4. The audio is sent to the OpenAI Whisper API, chunked if it exceeds 24 MB.
func (w *whisperLayer) Transcribe(ctx context.Context, youtubeURL string) (*Result, error) {
	if w.apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY not configured")
	}
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

	// Step 2: normalise to compact 16 kHz mono MP3
	normAudio := filepath.Join(tmpDir, "audio_norm.mp3")
	if err := normalizeAudio(ctx, rawAudio, normAudio); err != nil {
		return nil, fmt.Errorf("ffmpeg normalize: %w", err)
	}

	// Step 3: transcribe (single shot or chunked)
	return w.transcribeFileChunked(ctx, tmpDir, normAudio)
}

// downloadAudio invokes yt-dlp to extract the best audio to a WAV file.
// Returns the path of the produced file.
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

	matches, err := filepath.Glob(filepath.Join(tmpDir, "audio.*"))
	if err != nil || len(matches) == 0 {
		return "", fmt.Errorf("yt-dlp produced no output file")
	}
	return matches[0], nil
}

// normalizeAudio re-encodes the input to 16 kHz mono 64 kbps MP3. Speech is
// intelligible at this bitrate and the small size keeps uploads under OpenAI's cap.
func normalizeAudio(ctx context.Context, input, output string) error {
	cmd := exec.CommandContext(ctx,
		"ffmpeg",
		"-i", input,
		"-ar", "16000", // 16 kHz sample rate
		"-ac", "1", // mono channel
		"-b:a", "64k", // 64 kbps — plenty for speech
		"-y", // overwrite output without asking
		"-loglevel", "error",
		output,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%w\nffmpeg output: %s", err, out)
	}
	return nil
}

// transcribeFileChunked transcribes normAudio, splitting into time-based chunks
// when the file exceeds the upload limit and stitching the results together with
// cumulative timestamp offsets.
func (w *whisperLayer) transcribeFileChunked(ctx context.Context, tmpDir, normAudio string) (*Result, error) {
	info, err := os.Stat(normAudio)
	if err != nil {
		return nil, fmt.Errorf("stat audio: %w", err)
	}

	if info.Size() <= openAIMaxUploadBytes {
		resp, err := w.transcribeOne(ctx, normAudio)
		if err != nil {
			return nil, err
		}
		return resp.toResult(0), nil
	}

	chunks, err := splitAudio(ctx, normAudio, tmpDir)
	if err != nil {
		return nil, fmt.Errorf("split audio: %w", err)
	}

	combined := &Result{WhisperUsed: true, Source: "whisper"}
	var offset float64
	for _, chunk := range chunks {
		resp, err := w.transcribeOne(ctx, chunk)
		if err != nil {
			return nil, fmt.Errorf("chunk %s: %w", filepath.Base(chunk), err)
		}
		part := resp.toResult(offset)
		if combined.Language == "" {
			combined.Language = part.Language
		}
		if combined.Text != "" && part.Text != "" {
			combined.Text += " "
		}
		combined.Text += part.Text
		combined.Segments = append(combined.Segments, part.Segments...)
		offset += resp.Duration
		combined.Duration = offset
	}
	return combined, nil
}

// splitAudio segments input into ~chunkSeconds MP3 files using ffmpeg's segment
// muxer. Returns the chunk paths in playback order.
func splitAudio(ctx context.Context, input, tmpDir string) ([]string, error) {
	pattern := filepath.Join(tmpDir, "chunk_%03d.mp3")
	cmd := exec.CommandContext(ctx,
		"ffmpeg",
		"-i", input,
		"-f", "segment",
		"-segment_time", fmt.Sprintf("%d", chunkSeconds),
		"-c", "copy",
		"-y",
		"-loglevel", "error",
		pattern,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("%w\nffmpeg output: %s", err, out)
	}

	matches, err := filepath.Glob(filepath.Join(tmpDir, "chunk_*.mp3"))
	if err != nil || len(matches) == 0 {
		return nil, fmt.Errorf("ffmpeg produced no chunks")
	}
	sort.Strings(matches) // %03d numbering keeps lexical == chronological order
	return matches, nil
}

// openAIResponse models OpenAI's verbose_json transcription payload.
type openAIResponse struct {
	Text     string  `json:"text"`
	Language string  `json:"language"`
	Duration float64 `json:"duration"`
	Segments []struct {
		Start float64 `json:"start"`
		End   float64 `json:"end"`
		Text  string  `json:"text"`
	} `json:"segments"`
}

// toResult converts an OpenAI response into a pipeline Result, shifting all
// segment timestamps by offset (used when stitching chunks).
func (r *openAIResponse) toResult(offset float64) *Result {
	segs := make([]Segment, len(r.Segments))
	for i, s := range r.Segments {
		segs[i] = Segment{
			Start: s.Start + offset,
			End:   s.End + offset,
			Text:  s.Text,
		}
	}
	return &Result{
		Text:        r.Text,
		Language:    r.Language,
		Duration:    r.Duration,
		WhisperUsed: true,
		Segments:    segs,
		Source:      "whisper",
	}
}

// transcribeOne uploads a single audio file to the OpenAI Whisper API and
// returns the decoded verbose_json response.
func (w *whisperLayer) transcribeOne(ctx context.Context, audioPath string) (*openAIResponse, error) {
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
	if err := mw.WriteField("model", "whisper-1"); err != nil {
		return nil, err
	}
	if err := mw.WriteField("response_format", "verbose_json"); err != nil {
		return nil, err
	}
	mw.Close()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, openAITranscribeURL, &body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+w.apiKey)
	req.Header.Set("Content-Type", mw.FormDataContentType())

	resp, err := w.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http post: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<10))
		return nil, fmt.Errorf("openai returned %d: %s", resp.StatusCode, raw)
	}

	var out openAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &out, nil
}
