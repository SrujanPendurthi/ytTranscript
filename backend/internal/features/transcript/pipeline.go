package transcript

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/redis/go-redis/v9"
)

type Segment struct {
	Start float64 `json:"start"`
	End   float64 `json:"end"`
	Text  string  `json:"text"`
}

// Result is the normalised output of all three pipeline layers.
type Result struct {
	Text        string    `json:"text"`
	Language    string    `json:"language"`
	Duration    float64   `json:"duration"`
	WhisperUsed bool      `json:"whisper_used"`
	Segments    []Segment `json:"segments"`
	// Source is "cache" | "captions" | "whisper" — useful for observability.
	Source string `json:"source"`
}

// Pipeline runs a 3-layer transcript extraction strategy.
// Layer 1: Redis cache.
// Layer 2: native YouTube closed-captions (no ffmpeg required).
// Layer 3: yt-dlp + ffmpeg audio download, then the OpenAI Whisper API.
type Pipeline struct {
	cache    *cacheLayer
	captions *captionsLayer
	whisper  *whisperLayer
}

func New(rdb *redis.Client, openAIKey string) *Pipeline {
	return &Pipeline{
		cache:    newCacheLayer(rdb),
		captions: newCaptionsLayer(),
		whisper:  newWhisperLayer(openAIKey),
	}
}

// Run executes layers in order and returns on the first success.
func (p *Pipeline) Run(ctx context.Context, youtubeURL string) (*Result, error) {
	videoID, err := ExtractVideoID(youtubeURL)
	if err != nil {
		return nil, fmt.Errorf("invalid youtube url: %w", err)
	}

	// Layer 1 — cache
	if r, err := p.cache.Get(ctx, videoID); err == nil && r != nil {
		return r, nil
	}

	// Layer 2 — native captions
	if r, err := p.captions.Extract(ctx, videoID); err == nil && r != nil {
		_ = p.cache.Set(ctx, videoID, r) // best-effort
		return r, nil
	}

	// Layer 3 — Whisper fallback
	r, err := p.whisper.Transcribe(ctx, youtubeURL)
	if err != nil {
		return nil, fmt.Errorf("all layers failed; whisper: %w", err)
	}
	_ = p.cache.Set(ctx, videoID, r)
	return r, nil
}

// ExtractVideoID parses a YouTube URL and returns the video ID.
// Supported formats: watch?v=, youtu.be/, /shorts/, /embed/.
func ExtractVideoID(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("parse url: %w", err)
	}

	switch strings.ToLower(u.Host) {
	case "youtu.be":
		id := strings.TrimPrefix(u.Path, "/")
		if id == "" {
			return "", fmt.Errorf("no video id in youtu.be url")
		}
		return id, nil

	case "www.youtube.com", "youtube.com", "m.youtube.com":
		// /watch?v=ID
		if id := u.Query().Get("v"); id != "" {
			return id, nil
		}
		// /shorts/ID  or  /embed/ID
		for _, prefix := range []string{"/shorts/", "/embed/"} {
			if strings.HasPrefix(u.Path, prefix) {
				id := strings.TrimPrefix(u.Path, prefix)
				if id != "" {
					return id, nil
				}
			}
		}
		return "", fmt.Errorf("no video id found in youtube url")

	default:
		return "", fmt.Errorf("unsupported host: %s", u.Host)
	}
}
