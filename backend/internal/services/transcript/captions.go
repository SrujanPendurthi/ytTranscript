package transcript

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// captionURLRe extracts the first captionTrack baseUrl from ytInitialPlayerResponse.
// YouTube embeds the JSON in a <script> tag; the URL uses JSON escape sequences
// like & (→ &) and \/ (→ /) which we unescape after extraction.
var captionURLRe = regexp.MustCompile(`"captionTracks":\s*\[{"baseUrl":"([^"]+)"`)

type captionsLayer struct {
	client *http.Client
}

func newCaptionsLayer() *captionsLayer {
	return &captionsLayer{
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

// Extract fetches native YouTube captions for videoID and returns plain text.
// Returns (nil, err) when captions are unavailable — caller falls through to Layer 3.
func (c *captionsLayer) Extract(ctx context.Context, videoID string) (*Result, error) {
	captionURL, err := c.fetchCaptionURL(ctx, videoID)
	if err != nil {
		return nil, fmt.Errorf("captions: locate url: %w", err)
	}

	lang, err := langFromURL(captionURL)
	if err != nil {
		lang = "unknown"
	}

	text, duration, err := c.fetchAndParseXML(ctx, captionURL)
	if err != nil {
		return nil, fmt.Errorf("captions: parse xml: %w", err)
	}
	if strings.TrimSpace(text) == "" {
		return nil, fmt.Errorf("captions: empty transcript")
	}

	return &Result{
		Text:        text,
		Language:    lang,
		Duration:    duration,
		WhisperUsed: false,
		Source:      "captions",
	}, nil
}

// fetchCaptionURL scrapes the YouTube watch page for the first captionTrack URL.
func (c *captionsLayer) fetchCaptionURL(ctx context.Context, videoID string) (string, error) {
	watchURL := "https://www.youtube.com/watch?v=" + videoID
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, watchURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) "+
		"AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 5<<20)) // 5 MB cap
	if err != nil {
		return "", err
	}

	m := captionURLRe.FindSubmatch(body)
	if m == nil {
		return "", fmt.Errorf("no captionTracks found in page (video may have no captions)")
	}

	// m[1] is the raw JSON string value — unescape JSON escape sequences.
	var captionURL string
	if err := json.Unmarshal([]byte(`"`+string(m[1])+`"`), &captionURL); err != nil {
		return "", fmt.Errorf("unescape caption url: %w", err)
	}
	return captionURL, nil
}

// captionXML matches YouTube's timedtext XML format.
type captionXML struct {
	XMLName xml.Name      `xml:"transcript"`
	Texts   []captionLine `xml:"text"`
}

type captionLine struct {
	Start   float64 `xml:"start,attr"`
	Dur     float64 `xml:"dur,attr"`
	Content string  `xml:",chardata"`
}

func (c *captionsLayer) fetchAndParseXML(ctx context.Context, rawURL string) (string, float64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", 0, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", 0, fmt.Errorf("timedtext returned %d", resp.StatusCode)
	}

	var doc captionXML
	if err := xml.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return "", 0, fmt.Errorf("decode xml: %w", err)
	}

	var sb strings.Builder
	var duration float64
	for i, line := range doc.Texts {
		text := html.UnescapeString(strings.TrimSpace(line.Content))
		// Strip newlines within a single caption segment.
		text = strings.ReplaceAll(text, "\n", " ")
		if text == "" {
			continue
		}
		if i > 0 {
			sb.WriteByte(' ')
		}
		sb.WriteString(text)
		end := line.Start + line.Dur
		if end > duration {
			duration = end
		}
	}

	return sb.String(), duration, nil
}

func langFromURL(captionURL string) (string, error) {
	u, err := url.Parse(captionURL)
	if err != nil {
		return "", err
	}
	if lang := u.Query().Get("lang"); lang != "" {
		return lang, nil
	}
	return "", fmt.Errorf("no lang param")
}
