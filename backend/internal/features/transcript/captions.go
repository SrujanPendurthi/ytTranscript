package transcript

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// innerTubeKey is the well-known public InnerTube "WEB" client API key. It is
// embedded in every youtube.com page and is not a secret.
const innerTubeKey = "AIzaSyAO_FJ2SlqU8Q4STEHLGCilw_Y9_11qcW8"

// captionURLRe extracts the first captionTrack baseUrl from ytInitialPlayerResponse.
// Used only by the HTML-scrape fallback when the InnerTube API returns no tracks.
// YouTube embeds the JSON in a <script> tag; the URL uses JSON escape sequences
// like & (→ &) and \/ (→ /) which we unescape after extraction.
var captionURLRe = regexp.MustCompile(`"captionTracks":\s*(\[.+?\])`)

type captionsLayer struct {
	client *http.Client
}

func newCaptionsLayer() *captionsLayer {
	return &captionsLayer{
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

// captionTrack is one available caption source for a video.
type captionTrack struct {
	BaseURL      string `json:"baseUrl"`
	LanguageCode string `json:"languageCode"`
	// Kind is "asr" for auto-generated tracks, empty for human-authored ones.
	Kind string `json:"kind"`
}

// Extract fetches native YouTube captions for videoID and returns plain text
// with timestamped segments. Returns (nil, err) when captions are unavailable —
// the caller falls through to the Whisper layer.
func (c *captionsLayer) Extract(ctx context.Context, videoID string) (*Result, error) {
	tracks, err := c.fetchCaptionTracks(ctx, videoID)
	if err != nil {
		return nil, fmt.Errorf("captions: list tracks: %w", err)
	}
	if len(tracks) == 0 {
		return nil, fmt.Errorf("captions: no caption tracks for video")
	}

	track := selectTrack(tracks)

	text, segments, duration, err := c.fetchTrack(ctx, track.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("captions: fetch track: %w", err)
	}
	if strings.TrimSpace(text) == "" {
		return nil, fmt.Errorf("captions: empty transcript")
	}

	lang := track.LanguageCode
	if lang == "" {
		lang = "unknown"
	}

	return &Result{
		Text:        text,
		Language:    lang,
		Duration:    duration,
		WhisperUsed: false,
		Segments:    segments,
		Source:      "captions",
	}, nil
}

// fetchCaptionTracks lists caption tracks, preferring the structured InnerTube
// player API and falling back to scraping the watch page.
func (c *captionsLayer) fetchCaptionTracks(ctx context.Context, videoID string) ([]captionTrack, error) {
	if tracks, err := c.fetchTracksInnerTube(ctx, videoID); err == nil && len(tracks) > 0 {
		return tracks, nil
	}
	// Fallback: scrape ytInitialPlayerResponse out of the watch page HTML.
	return c.fetchTracksFromHTML(ctx, videoID)
}

// playerResponse is the subset of the InnerTube /player response we care about.
type playerResponse struct {
	Captions struct {
		PlayerCaptionsTracklistRenderer struct {
			CaptionTracks []captionTrack `json:"captionTracks"`
		} `json:"playerCaptionsTracklistRenderer"`
	} `json:"captions"`
}

// fetchTracksInnerTube calls the InnerTube /player endpoint — a stable JSON API
// that does not depend on scraping page markup.
func (c *captionsLayer) fetchTracksInnerTube(ctx context.Context, videoID string) ([]captionTrack, error) {
	body := map[string]any{
		"videoId": videoID,
		"context": map[string]any{
			"client": map[string]any{
				"clientName":    "WEB",
				"clientVersion": "2.20240101.00.00",
			},
		},
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	url := "https://www.youtube.com/youtubei/v1/player?key=" + innerTubeKey
	raw, err := c.doWithRetry(ctx, func() (*http.Request, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", desktopUA)
		req.Header.Set("Accept-Language", "en-US,en;q=0.9")
		return req, nil
	})
	if err != nil {
		return nil, err
	}

	var pr playerResponse
	if err := json.Unmarshal(raw, &pr); err != nil {
		return nil, fmt.Errorf("decode player response: %w", err)
	}
	return pr.Captions.PlayerCaptionsTracklistRenderer.CaptionTracks, nil
}

// fetchTracksFromHTML scrapes the watch page for the captionTracks JSON array.
func (c *captionsLayer) fetchTracksFromHTML(ctx context.Context, videoID string) ([]captionTrack, error) {
	watchURL := "https://www.youtube.com/watch?v=" + videoID
	raw, err := c.doWithRetry(ctx, func() (*http.Request, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, watchURL, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", desktopUA)
		req.Header.Set("Accept-Language", "en-US,en;q=0.9")
		return req, nil
	})
	if err != nil {
		return nil, err
	}

	m := captionURLRe.FindSubmatch(raw)
	if m == nil {
		return nil, fmt.Errorf("no captionTracks found in page (video may have no captions)")
	}

	// m[1] is a JSON array; the embedded baseUrl values use & for & which
	// standard JSON decoding handles. Trailing keys we don't model are ignored.
	var tracks []captionTrack
	if err := json.Unmarshal(m[1], &tracks); err != nil {
		return nil, fmt.Errorf("decode captionTracks: %w", err)
	}
	return tracks, nil
}

// selectTrack picks the best caption track: manual English → any English →
// any manual → first available.
func selectTrack(tracks []captionTrack) captionTrack {
	isEN := func(t captionTrack) bool { return strings.HasPrefix(strings.ToLower(t.LanguageCode), "en") }
	isManual := func(t captionTrack) bool { return t.Kind != "asr" }

	for _, t := range tracks {
		if isEN(t) && isManual(t) {
			return t
		}
	}
	for _, t := range tracks {
		if isEN(t) {
			return t
		}
	}
	for _, t := range tracks {
		if isManual(t) {
			return t
		}
	}
	return tracks[0]
}

// fetchTrack downloads a caption track and returns text, segments, and duration.
// It requests the json3 format first and falls back to the legacy XML format.
func (c *captionsLayer) fetchTrack(ctx context.Context, baseURL string) (string, []Segment, float64, error) {
	// json3 gives structured events with millisecond timing.
	if text, segs, dur, err := c.fetchJSON3(ctx, baseURL); err == nil && strings.TrimSpace(text) != "" {
		return text, segs, dur, nil
	}
	// Fallback: the default timedtext XML format.
	return c.fetchXML(ctx, baseURL)
}

// json3Response models YouTube's fmt=json3 timedtext payload.
type json3Response struct {
	Events []struct {
		TStartMs    int64 `json:"tStartMs"`
		DDurationMs int64 `json:"dDurationMs"`
		Segs        []struct {
			Utf8 string `json:"utf8"`
		} `json:"segs"`
	} `json:"events"`
}

func (c *captionsLayer) fetchJSON3(ctx context.Context, baseURL string) (string, []Segment, float64, error) {
	raw, err := c.doWithRetry(ctx, func() (*http.Request, error) {
		return http.NewRequestWithContext(ctx, http.MethodGet, appendFormat(baseURL, "json3"), nil)
	})
	if err != nil {
		return "", nil, 0, err
	}
	return parseJSON3(raw)
}

// parseJSON3 decodes a fmt=json3 timedtext body into text, segments, and duration.
func parseJSON3(raw []byte) (string, []Segment, float64, error) {
	var doc json3Response
	if err := json.Unmarshal(raw, &doc); err != nil {
		return "", nil, 0, fmt.Errorf("decode json3: %w", err)
	}

	var sb strings.Builder
	var segments []Segment
	var duration float64
	for _, ev := range doc.Events {
		var line strings.Builder
		for _, s := range ev.Segs {
			line.WriteString(s.Utf8)
		}
		text := strings.TrimSpace(strings.ReplaceAll(line.String(), "\n", " "))
		if text == "" {
			continue
		}
		start := float64(ev.TStartMs) / 1000
		end := float64(ev.TStartMs+ev.DDurationMs) / 1000
		segments = append(segments, Segment{Start: start, End: end, Text: text})
		if end > duration {
			duration = end
		}
		if sb.Len() > 0 {
			sb.WriteByte(' ')
		}
		sb.WriteString(text)
	}
	return sb.String(), segments, duration, nil
}

// captionXML matches YouTube's legacy timedtext XML format.
type captionXML struct {
	XMLName xml.Name      `xml:"transcript"`
	Texts   []captionLine `xml:"text"`
}

type captionLine struct {
	Start   float64 `xml:"start,attr"`
	Dur     float64 `xml:"dur,attr"`
	Content string  `xml:",chardata"`
}

func (c *captionsLayer) fetchXML(ctx context.Context, baseURL string) (string, []Segment, float64, error) {
	raw, err := c.doWithRetry(ctx, func() (*http.Request, error) {
		return http.NewRequestWithContext(ctx, http.MethodGet, baseURL, nil)
	})
	if err != nil {
		return "", nil, 0, err
	}
	return parseXML(raw)
}

// parseXML decodes a legacy timedtext XML body into text, segments, and duration.
func parseXML(raw []byte) (string, []Segment, float64, error) {
	var doc captionXML
	if err := xml.NewDecoder(bytes.NewReader(raw)).Decode(&doc); err != nil {
		return "", nil, 0, fmt.Errorf("decode xml: %w", err)
	}

	var sb strings.Builder
	var segments []Segment
	var duration float64
	for _, line := range doc.Texts {
		text := html.UnescapeString(strings.TrimSpace(line.Content))
		text = strings.ReplaceAll(text, "\n", " ")
		if text == "" {
			continue
		}
		end := line.Start + line.Dur
		segments = append(segments, Segment{Start: line.Start, End: end, Text: text})
		if end > duration {
			duration = end
		}
		if sb.Len() > 0 {
			sb.WriteByte(' ')
		}
		sb.WriteString(text)
	}
	return sb.String(), segments, duration, nil
}

const desktopUA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) " +
	"AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

// appendFormat adds &fmt=<f> (or ?fmt=<f>) to a timedtext base URL.
func appendFormat(baseURL, f string) string {
	sep := "&"
	if !strings.Contains(baseURL, "?") {
		sep = "?"
	}
	return baseURL + sep + "fmt=" + f
}

// doWithRetry executes the request built by newReq up to 3 times, retrying on
// transport errors and 429/5xx responses with a short linear backoff. It returns
// the response body (capped at 10 MB) on the first 2xx.
func (c *captionsLayer) doWithRetry(ctx context.Context, newReq func() (*http.Request, error)) ([]byte, error) {
	const maxAttempts = 3
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		req, err := newReq()
		if err != nil {
			return nil, err
		}

		resp, err := c.client.Do(req)
		if err != nil {
			lastErr = err
		} else {
			body, readErr := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
			resp.Body.Close()
			switch {
			case readErr != nil:
				lastErr = readErr
			case resp.StatusCode >= 200 && resp.StatusCode < 300:
				return body, nil
			case resp.StatusCode == 429 || resp.StatusCode >= 500:
				lastErr = fmt.Errorf("status %d", resp.StatusCode)
			default:
				return nil, fmt.Errorf("status %d", resp.StatusCode)
			}
		}

		if attempt < maxAttempts {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(time.Duration(attempt) * 500 * time.Millisecond):
			}
		}
	}
	return nil, fmt.Errorf("request failed after %d attempts: %w", maxAttempts, lastErr)
}
