//go:build e2e

// Package e2e holds black-box tests that exercise a *running* TranscriptFlow
// stack over HTTP. They are excluded from the normal `go test ./...` run by the
// `e2e` build tag.
//
//	Run against a live stack (see docker compose up -d):
//	  cd backend && E2E_BASE_URL=http://localhost:8080 go test -tags e2e ./e2e/ -v
package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"
)

func baseURL() string {
	if v := os.Getenv("E2E_BASE_URL"); v != "" {
		return v
	}
	return "http://localhost:8080"
}

type reqOpts struct {
	method  string
	path    string
	body    any
	headers map[string]string
}

// do performs an HTTP request, decodes JSON into out (when non-nil), and returns
// the status code.
func do(t *testing.T, o reqOpts, out any) int {
	t.Helper()

	var body io.Reader
	if o.body != nil {
		b, err := json.Marshal(o.body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		body = bytes.NewReader(b)
	}

	req, err := http.NewRequest(o.method, baseURL()+o.path, body)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if o.body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range o.headers {
		req.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", o.method, o.path, err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if out != nil && len(raw) > 0 {
		if err := json.Unmarshal(raw, out); err != nil {
			t.Fatalf("decode %s %s (%d): %v\nbody: %s", o.method, o.path, resp.StatusCode, err, raw)
		}
	}
	return resp.StatusCode
}

// TestAPIKeyTranscribePath proves a developer API key minted through the
// dashboard endpoints can authenticate the public developer endpoints:
// POST /v1/transcribe and GET /v1/jobs/{id}.
func TestAPIKeyTranscribePath(t *testing.T) {
	email := fmt.Sprintf("e2e+%d@test.dev", time.Now().UnixNano())
	const sampleURL = "https://youtube.com/watch?v=dQw4w9WgXcQ"

	// 1. Sign up -> JWT (dashboard auth).
	var signup struct {
		Token string `json:"token"`
	}
	if code := do(t, reqOpts{method: http.MethodPost, path: "/api/auth/signup", body: map[string]string{
		"email": email, "password": "password123",
	}}, &signup); code != http.StatusCreated {
		t.Fatalf("signup: want 201, got %d", code)
	}
	if signup.Token == "" {
		t.Fatal("signup returned an empty token")
	}
	bearer := map[string]string{"Authorization": "Bearer " + signup.Token}

	// 2. Create an API key with the JWT.
	var key struct {
		ID     string `json:"id"`
		RawKey string `json:"raw_key"`
	}
	if code := do(t, reqOpts{method: http.MethodPost, path: "/api/v1/keys", body: map[string]string{
		"name": "e2e-test",
	}, headers: bearer}, &key); code != http.StatusCreated {
		t.Fatalf("create key: want 201, got %d", code)
	}
	if key.RawKey == "" {
		t.Fatal("create key returned an empty raw_key")
	}
	t.Cleanup(func() {
		do(t, reqOpts{method: http.MethodDelete, path: "/api/v1/keys/" + key.ID, headers: bearer}, nil)
	})
	apiKey := map[string]string{"X-API-Key": key.RawKey}

	// 3. Submit a job using ONLY the API key.
	var job struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	if code := do(t, reqOpts{method: http.MethodPost, path: "/api/v1/transcribe", body: map[string]string{
		"youtube_url": sampleURL,
	}, headers: apiKey}, &job); code != http.StatusAccepted {
		t.Fatalf("transcribe with API key: want 202, got %d", code)
	}
	if job.ID == "" {
		t.Fatal("transcribe returned an empty job id")
	}
	t.Logf("job %s accepted via API key (status=%s)", job.ID, job.Status)

	// 4. Fetch that job with the API key.
	var fetched struct {
		ID string `json:"id"`
	}
	if code := do(t, reqOpts{method: http.MethodGet, path: "/api/v1/jobs/" + job.ID, headers: apiKey}, &fetched); code != http.StatusOK {
		t.Fatalf("get job with API key: want 200, got %d", code)
	}
	if fetched.ID != job.ID {
		t.Fatalf("get job returned id %q, want %q", fetched.ID, job.ID)
	}

	// 5. Negative: no credentials -> 401.
	if code := do(t, reqOpts{method: http.MethodPost, path: "/api/v1/transcribe", body: map[string]string{
		"youtube_url": sampleURL,
	}}, nil); code != http.StatusUnauthorized {
		t.Fatalf("transcribe without auth: want 401, got %d", code)
	}

	// 6. Negative: a bogus key -> 401.
	if code := do(t, reqOpts{method: http.MethodPost, path: "/api/v1/transcribe", body: map[string]string{
		"youtube_url": sampleURL,
	}, headers: map[string]string{"X-API-Key": "tf_deadbeef"}}, nil); code != http.StatusUnauthorized {
		t.Fatalf("transcribe with bad key: want 401, got %d", code)
	}
}
