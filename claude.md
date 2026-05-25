# TranscriptFlow CLAUDE.md

## Project Vision & Architecture
TranscriptFlow is a YouTube transcript extraction SaaS built with a Go (Chi) backend, React (Vite+TS+Tailwind) frontend, PostgreSQL, Redis queues, and a Faster-Whisper audio-processing fallback worker.

## Build and Test Commands
- **Backend Build:** `cd backend && go build ./...`
- **Backend Test:** `cd backend && go test ./...`
- **Frontend Build:** `cd frontend && npm run build`
- **Frontend Test/Lint:** `cd frontend && npm run lint`
- **Docker Dev Environment:** `docker compose up -d`

## Code Style & Guardrails
### Go Backend:
- Use explicit type-hinting, strict dependency injection, and clean separation between handlers, repositories, and services.
- Never use raw `panic` in production paths; errors must be bubbles up and handled gracefully via Chi middleware.
- Secure all raw or parameterized SQL strings against injection using prepared statements via GORM or native pgx.

### React Frontend:
- Use functional TypeScript components with Vite, Tailwind CSS, and shadcn/ui.
- Maintain application state globally using Zustand and server state via React Query.
- Keep components modular; separate layout views from presentation wrappers.

## Implementation Roadmap (State Machine)
- [ ] Phase 1: Directory Scaffolding & Shared Docker Setup
- [ ] Phase 2: PostgreSQL Tables & GORM Database Migrations
- [ ] Phase 3: JWT Web Auth & Hashed Developer API Key Middleware
- [ ] Phase 4: Redis Cache Connection & Async Asynq Work Queue Layout
- [ ] Phase 5: Go Transcript Scraper Engine & Audio Downloader Pipeline
- [ ] Phase 6: Python Faster-Whisper Container & IPC/Network Endpoint
- [ ] Phase 7: Stripe Checkout Link Generation & Signature Webhook Handler
- [ ] Phase 8: Component Design System, Dashboard, and Dev Portal in React
- [ ] Phase 9: End-to-End Orchestration & .env Setup Validation

Sacrifice grammar and speak to me concisly. Think a lot before you act. Ask any questions you may have.

