# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Vision
TranscriptFlow is a YouTube transcript extraction SaaS: Go (Chi) API + async worker, React (Vite+TS+Tailwind) frontend, PostgreSQL, Redis, and the OpenAI Whisper API as an audio-transcription fallback. All 9 roadmap phases below are implemented; treat them as a map of the system, not a to-do list.

## Build, Lint, Test, Run
- **Backend build:** `cd backend && go build ./...`
- **Backend test:** `cd backend && go test ./...` — single test: `go test ./internal/features/transcript/ -run TestParseJSON3 -v`
- **Backend e2e (needs a running stack):** `cd backend && E2E_BASE_URL=http://localhost:8080 go test -tags e2e ./e2e/ -v` — black-box HTTP test proving the API-key → `/v1/transcribe` path; excluded from the normal suite by the `e2e` build tag.
- **Frontend build:** `cd frontend && npm run build` (runs `tsc` then `vite build`)
- **Frontend lint:** `cd frontend && npm run lint` (eslint 9 flat config, `--max-warnings 0` — warnings fail)
- **Frontend dev server:** `cd frontend && npm run dev`
- **Full stack:** `docker compose up -d` (postgres, redis, backend, worker, frontend). Set `OPENAI_API_KEY` in `.env` for the Whisper fallback.
- Go 1.26; module path `github.com/srujanpendurthi/transcriptflow`. `backend/vendor/` is checked in — build offline with `go build -mod=vendor ./...`.

## Project layout — feature-based
Both stacks group code by domain feature, with cross-cutting entities/infra in a shared package. **When adding code, put it in the owning feature; only truly shared entities/infra go in `platform` (Go) or `shared` (React).**

- `backend/internal/features/{auth,transcript,billing,apikey}/` — each owns its `handler.go` + feature logic (transcript also holds `pipeline/cache/captions/whisper/task/processor`; auth & apikey hold their middleware).
- `backend/internal/platform/{config,db,queue,models,utils}/` — shared infra + GORM entities. `platform/queue` is generic asynq/redis plumbing; feature-specific tasks live in the feature (e.g. `transcript/task.go`).
- `frontend/src/features/{auth,transcripts,developer}/` — each a bundle of `api/ components/ hooks/ types/` plus an `index.ts` **public API**; import a feature via its barrel (`@/features/auth`), not deep paths. Global buckets at `src/`: `config/` (axios client + `ApiError`), `components/` (global UI like `AppLayout`), `utils/`, `context/`, `hooks/`, `assets/`.

## Architecture — the big picture

**Two Go binaries, one module.** `cmd/api` (HTTP server) and `cmd/worker` (asynq consumer) share `internal/`. The API only *enqueues*; the worker does all transcript extraction. They communicate solely through Postgres (job rows) + Redis (asynq queue) — never call worker code from the API.

**Request → job lifecycle:**
1. `POST /api/v1/transcribe` (`features/transcript/handler.go`) validates the URL, writes a `pending` `TranscriptionJob`, enqueues an asynq task carrying **only the job UUID**, returns `202`. If enqueue fails it deletes the row so nothing gets stuck.
2. Worker's `TranscriptionProcessor.ProcessTask` (`features/transcript/processor.go`) owns the state machine: `pending → processing → completed | failed`. It re-fetches the full job from Postgres (the queue payload is just the ID) and is idempotent (skips already-`completed` jobs — asynq retries up to 3×).
3. Client polls `GET /api/v1/jobs/{id}` until terminal status.

**3-layer extraction pipeline** (`internal/features/transcript/pipeline.go`), tried in order, first success wins; `Result.Source` records which layer won (`cache`/`captions`/`whisper`) and is persisted on the job:
- Layer 1 — Redis cache keyed by video ID (`cache.go`)
- Layer 2 — native YouTube captions (`captions.go`): lists tracks via the InnerTube `/youtubei/v1/player` API (HTML-scrape fallback), fetches json3 (XML fallback), populates timestamped segments. NOTE: from datacenter IPs YouTube often returns empty caption bodies — the layer then errors and falls through to Whisper (expected).
- Layer 3 — Whisper fallback (`whisper.go`): `yt-dlp` → `ffmpeg` (compact 16 kHz mono MP3) → multipart POST to the **OpenAI** `/v1/audio/transcriptions` (`whisper-1`, `verbose_json`); files >24 MB are chunked and segment timestamps offset-stitched. Requires `yt-dlp` + `ffmpeg` on PATH (worker image) and `OPENAI_API_KEY`.
- `ExtractVideoID` handles `watch?v=`, `youtu.be/`, `/shorts/`, `/embed/` — reuse it, don't re-parse URLs.
- `whisper-service/` (Python Faster-Whisper) is legacy/dead code — no longer wired into compose or the pipeline.

**Auth — middlewares live with their features:**
- `auth.JWT` (`features/auth/middleware.go`) — web dashboard (Bearer token, `features/auth/authStore.ts` persists it to localStorage as `tf-auth`).
- `apikey.APIKey` (`features/apikey/middleware.go`) — developer clients via `X-API-Key`; the raw key is SHA-256 hashed and looked up against the stored hash (raw key is never stored). Bumps `last_used_at`.
- `apikey.JWTOrAPIKey` (`features/apikey/dual_auth.go`) — used by `/transcribe` and `/jobs/{id}` so both callers work: presence of `Authorization: Bearer ` picks JWT, else API key. When adding an endpoint that both dashboard and developers need, wire it under this group in `cmd/api/main.go`.
- Identity flows via context (`features/auth/context.go`): `UserIDFrom`, `APIKeyIDFrom`. Handlers must scope all queries by `user_id` for ownership.

**Billing:** Stripe checkout + webhook (`features/billing/handler.go`). Webhook is public but signature-validated inside the handler; `stripe.Key` is set once in `main.go`. Plans: `free` / `pro` / `enterprise` on `User.Plan`.

**DB:** GORM auto-migrate on API startup (`platform/db/migrate.go`), order: User → APIKey → UsageLog → TranscriptionJob. Models embed `models.Base`. No separate migration files — schema changes go in struct tags.

**Frontend:** React Query for server state, Zustand for auth state (`features/auth/hooks/authStore.ts`). `config/api.ts` is the axios instance — a request interceptor injects the JWT at call-time (always fresh), a response interceptor hard-redirects to `/login` on any `401`. It imports the auth store leaf directly (not the feature barrel) to keep the core client free of feature UI. nginx proxies `/api/*` to the backend, so all routes live under `/api`.

## Code Style & Guardrails
### Go
- Keep features self-contained; use explicit types and dependency injection (constructors like `transcript.NewHandler(db, queue)`). Feature packages may import `platform`, never vice versa.
- No raw `panic` in request paths — bubble errors up; `chimw.Recoverer` is the safety net, not the strategy.
- All SQL through GORM / parameterized queries — never string-concatenate user input.

### React
- Functional TS components, Tailwind + shadcn/ui. Zustand for global/app state, React Query for server state. New feature code goes in `features/<feature>/{api,components,hooks,types}` and is exported via that feature's `index.ts`; only truly global UI/infra lives in the top-level `components/ config/ utils/ hooks/ context/`.

## Roadmap (all implemented — reference map)
1. Directory scaffolding & Docker · 2. Postgres + GORM migrations · 3. JWT + hashed API-key middleware · 4. Redis cache + asynq queue · 5. Go scraper + audio downloader pipeline · 6. Faster-Whisper container + endpoint · 7. Stripe checkout + webhook · 8. React design system, dashboard, dev portal · 9. E2E orchestration + `.env` (`.env.example` is the template)

---
Sacrifice grammar and speak to me concisely. Think a lot before you act. Ask any questions you may have.
