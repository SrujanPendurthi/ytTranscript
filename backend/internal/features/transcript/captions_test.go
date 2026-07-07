package transcript

import (
	"math"
	"testing"
)

func TestParseJSON3(t *testing.T) {
	raw := []byte(`{
	  "events": [
	    {"tStartMs": 0,    "dDurationMs": 1500, "segs": [{"utf8": "Hello"}, {"utf8": " world"}]},
	    {"tStartMs": 1500, "dDurationMs": 500,  "segs": [{"utf8": "\n"}]},
	    {"tStartMs": 2000, "dDurationMs": 1000, "segs": [{"utf8": "again"}]}
	  ]
	}`)

	text, segs, dur, err := parseJSON3(raw)
	if err != nil {
		t.Fatalf("parseJSON3: %v", err)
	}
	if text != "Hello world again" {
		t.Errorf("text = %q, want %q", text, "Hello world again")
	}
	// The whitespace-only middle event is dropped, leaving two real segments.
	if len(segs) != 2 {
		t.Fatalf("segments = %d, want 2", len(segs))
	}
	if segs[0].Start != 0 || !floatEq(segs[0].End, 1.5) {
		t.Errorf("segment[0] timing = %+v", segs[0])
	}
	if !floatEq(segs[1].Start, 2.0) || !floatEq(segs[1].End, 3.0) {
		t.Errorf("segment[1] timing = %+v", segs[1])
	}
	if !floatEq(dur, 3.0) {
		t.Errorf("duration = %v, want 3.0", dur)
	}
}

func TestParseXML(t *testing.T) {
	raw := []byte(`<?xml version="1.0" encoding="utf-8"?>
	<transcript>
	  <text start="0" dur="2.5">Hello &amp; welcome</text>
	  <text start="2.5" dur="1.5">to the show</text>
	</transcript>`)

	text, segs, dur, err := parseXML(raw)
	if err != nil {
		t.Fatalf("parseXML: %v", err)
	}
	if text != "Hello & welcome to the show" {
		t.Errorf("text = %q", text)
	}
	if len(segs) != 2 {
		t.Fatalf("segments = %d, want 2", len(segs))
	}
	if !floatEq(dur, 4.0) {
		t.Errorf("duration = %v, want 4.0", dur)
	}
}

func TestParseJSON3_NoContent(t *testing.T) {
	// A player response with no usable caption events yields empty text so the
	// pipeline can fall through to the Whisper layer.
	text, segs, _, err := parseJSON3([]byte(`{"events": []}`))
	if err != nil {
		t.Fatalf("parseJSON3: %v", err)
	}
	if text != "" || len(segs) != 0 {
		t.Errorf("expected empty result, got text=%q segs=%d", text, len(segs))
	}
}

func TestSelectTrack(t *testing.T) {
	cases := []struct {
		name   string
		tracks []captionTrack
		want   string // expected BaseURL
	}{
		{
			name: "prefers manual english over auto english",
			tracks: []captionTrack{
				{BaseURL: "auto-en", LanguageCode: "en", Kind: "asr"},
				{BaseURL: "manual-en", LanguageCode: "en"},
			},
			want: "manual-en",
		},
		{
			name: "prefers any english over manual non-english",
			tracks: []captionTrack{
				{BaseURL: "manual-fr", LanguageCode: "fr"},
				{BaseURL: "auto-en", LanguageCode: "en", Kind: "asr"},
			},
			want: "auto-en",
		},
		{
			name: "prefers manual when no english",
			tracks: []captionTrack{
				{BaseURL: "auto-de", LanguageCode: "de", Kind: "asr"},
				{BaseURL: "manual-fr", LanguageCode: "fr"},
			},
			want: "manual-fr",
		},
		{
			name: "falls back to first",
			tracks: []captionTrack{
				{BaseURL: "auto-de", LanguageCode: "de", Kind: "asr"},
				{BaseURL: "auto-fr", LanguageCode: "fr", Kind: "asr"},
			},
			want: "auto-de",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := selectTrack(tc.tracks).BaseURL; got != tc.want {
				t.Errorf("selectTrack = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestAppendFormat(t *testing.T) {
	if got := appendFormat("https://x/api/timedtext?v=1", "json3"); got != "https://x/api/timedtext?v=1&fmt=json3" {
		t.Errorf("with query: %q", got)
	}
	if got := appendFormat("https://x/api/timedtext", "json3"); got != "https://x/api/timedtext?fmt=json3" {
		t.Errorf("no query: %q", got)
	}
}

func floatEq(a, b float64) bool { return math.Abs(a-b) < 1e-9 }
