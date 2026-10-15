# Menyayka

[English](README.md) | [Русский](README.ru.md)

A web service for multi-party barter: a user lists an item and describes what they want in return; the system finds circular exchange chains among several participants (2–5 people), suggests options, collects responses and completes the deal through soft and hard locking of offers.

Demo: https://menyayka.lorot.dev

---

## Table of contents

1. [Project overview](#project-overview)
   - [Exchange safety](#exchange-safety)
   - [Ranking (LightGBM)](#ranking-lightgbm)
2. [Getting started](#getting-started)
   - [Whole project in one command](#whole-project-in-one-command)
   - [Frontend development without Docker](#frontend-development-without-docker-optional)
   - [Useful commands](#useful-commands)
   - [CI/CD and deploy](#cicd-and-deploy)
3. [Tech stack](#tech-stack)
4. [Project structure](#project-structure)
5. [Backend](#backend)
   - [Embeddings (TEI)](#embeddings-tei)
   - [Realtime (WebSocket)](#realtime-websocket)
   - [API reference](#api-reference)
6. [Frontend](#frontend)
   - [Realtime (WebSocket)](#realtime-websocket-1)
7. [Team](#team)

---

## Project overview

Key user scenarios:

- sign up / sign in / password recovery;
- personal item catalog;
- exchange offers (what I give + what I want);
- browsing discovered candidate chains;
- fast participant replacement on refusal;
- tracking shipping and receiving.

How matching and chains work:

1. An offer gets embeddings (offer + want) via TEI (or a stub in tests).
2. Similar offers are grouped into clusters (cosine + category filtering + direction margin).
3. A graph is built between clusters; DFS finds simple cycles of length 2–5.
4. Discovered cycles are stored as `CANDIDATE`, deduplicated and scored.
5. Participants cast directed votes; once the ring closes, the chain → `PROPOSED` (soft lock, ~24h deadline).
6. A second confirmation round → `FROZEN` (hard lock of items/offers).
7. On refusal — rollback or a fast replacement from the same cluster.
8. Then shipping/receiving tracking until the exchange is complete.

### Exchange safety

So that a chain participant doesn't give away their item and end up without the promised item in return, the exchange goes through a pickup point (PVZ) using a group storage model.

After the chain is confirmed, each participant must deliver their item to the pickup point by a set deadline. On acceptance, the item is checked against the offer's characteristics. Until all items are checked, everything is stored at the pickup point and not handed over to the new owners.

The chain is ready for handover only when all participants have delivered their items and every item has passed inspection. After that, all items become available to the recipients at once.

If someone fails to deliver on time or an item doesn't match the offer, the chain is cancelled. Items already accepted at the pickup point are returned to their original owners. This way nobody loses their item until the whole chain is confirmed to be completable.

### Ranking (LightGBM)

Candidates are ranked by the probability of a successful completion. A linear formula set that order before; a LightGBM model replaced it. The formula ignored the weak-edge threshold, liquidity saturation and a penalty for chain length that grows faster than linearly.

The model is trained on a synthetic simulator (`ml/simulator`) that reproduces the full deal funnel. Its output is a *calibrated* success probability.

Simulation is the only option at cold start: there is no history of real exchanges. The simulator encodes domain laws into latent variables; the model sees only noisy observable features, the same ones that run in production. There is no target leakage, and `ml/features.json` mirrors the Go feature extractor.

---

## Getting started

### Whole project in one command

Requires Docker + Docker Compose (Make is optional; you can call `docker compose` directly).

```bash
cp .env.example .env   # on Apple Silicon add TEI_IMAGE=...cpu-arm64-latest (see .env.example)
docker compose up -d --build
# or: make up
```

This brings up everything: PostgreSQL + pgvector, migrations, SeaweedFS (S3-compatible object storage), TEI, backend and frontend.

| What | URL |
|------|-----|
| UI (frontend) | http://localhost:3000 |
| API (backend) | http://localhost:8080 |
| SeaweedFS S3 | http://localhost:9000 |

The frontend serves static files and proxies `/api` → backend, `/s3` → SeaweedFS (same-origin, empty `VITE_API_BASE_URL`).

Stop:

```bash
make down        # containers + DB/SeaweedFS data; TEI cache is kept
make down-all    # full reset, including the model cache
```

### Frontend development without Docker (optional)

```bash
docker compose up -d db migrate seaweedfs tei app
cd frontend && npm ci && npm run dev   # http://localhost:5173 → proxy to :8080
```

### Running on ARM (Apple Silicon, ARM64 VPS, Raspberry Pi)

Everything runs natively on `linux/arm64` except the embeddings service, whose
`huggingface/text-embeddings-inference:cpu-1.9` tag only ships `linux/amd64`
(no multi-arch manifest, no versioned `arm64` tag). Set the arm-specific tag
before bringing the stack up:

```bash
echo 'TEI_IMAGE=ghcr.io/huggingface/text-embeddings-inference:cpu-arm64-latest' >> .env
make up
```

All other images (`pgvector`, `chrislusf/seaweedfs`, `migrate/migrate`,
`caddy:2-alpine`, `node:22-alpine`, `golang:1.26-alpine`) ship multi-arch and
are selected automatically by Docker.

### Useful commands

```bash
make logs            # backend + frontend logs
make db-logs         # database logs
make test            # backend unit tests
make run             # run backend on the host (needs a running DB)
make test-e2e        # API E2E (light compose without TEI)
make loadtest        # k6: register → item → list → login
```

### CI/CD and deploy

On pull requests, GitHub Actions runs backend unit tests, frontend typecheck/lint/test and E2E. On push to `main`, the same checks plus mandatory E2E run before the images are built for GHCR and deployed.

Production stack: `docker-compose.prod.yml` + Caddy + images from GHCR. Auto-deploy to a VPS on push to `main` (workflow `.github/workflows/deploy.yml`): image build and push to GHCR → SSH deploy to VPS (migrate + `docker compose up`) → Caddy serves your domain (set `Caddyfile` `server_name` and `WS_ALLOWED_ORIGINS` in `.env`).

---

## Tech stack

### Backend

| Component | Technology |
|-----------|------------|
| Language | Go 1.26 |
| HTTP | Gin |
| Database | PostgreSQL 18 + pgvector 0.8.6 |
| Migrations | golang-migrate |
| Auth | JWT (Bearer) |
| Embeddings | Hugging Face TEI (`multilingual-e5-small`) / stub |
| Object storage | SeaweedFS (S3-compatible) |
| Email | SMTP (password recovery) |
| Tests | `testing`, httptest; E2E with build tag `e2e` |
| Realtime | WebSocket (`github.com/coder/websocket`), in-memory hub |
| Load testing | k6 |

### Frontend

| Component | Technology |
|-----------|------------|
| UI | React 19 + TypeScript |
| Build | Vite |
| Architecture | Feature-Sliced Design (FSD) |
| State | Redux Toolkit + TanStack Query |
| UI kit | Ant Design |
| Routing | React Router |
| HTTP | Axios |
| Quality | ESLint, Prettier, Vitest, Testing Library |
| Realtime | native WebSocket API (fallback — polling) |

### Infrastructure

| Component | Technology |
|-----------|------------|
| Local run | Docker Compose, Makefile |
| Reverse proxy / TLS | Caddy |

---

## Project structure

```
menyayka/
├── backend/
│   ├── cmd/api/                 # composition root, HTTP server
│   ├── internal/
│   │   ├── api/                 # unified response/error format
│   │   ├── core/                # config, router, database, logger
│   │   ├── entity/              # domain entities and sentinel errors
│   │   ├── handler/             # HTTP layer (user, item, exchange_offer, chain)
│   │   ├── middleware/          # JWT auth
│   │   ├── realtime/            # WebSocket hub, tickets, event fan-out
│   │   ├── repository/          # DB access per feature (chain, cluster, item,
│   │   │                        #   search, exchange_offer, user, …)
│   │   ├── service/             # business logic (matching, clustering, chain, …)
│   │   └── infrastructure/      # TEI, mailer, reservation, chain notification
│   ├── migrations/              # SQL up/down (golang-migrate)
│   ├── docs/                    # generated Swagger spec + docs.go (swag init)
│   ├── pkg/                     # token, storage, validator, codestore, ranker
│   │   └── utils/ranker/models/ # exported LightGBM model (text v3) + Go engine
│   └── tests/                   # unit / handler / service / repository / e2e
├── frontend/
│   ├── src/
│   │   ├── app/                 # providers, router, store, themes
│   │   ├── pages/               # screens
│   │   ├── features/            # auth, items, exchange-requests, chains, deal
│   │   ├── entities/            # entity models and APIs
│   │   └── shared/              # UI, api client, realtime client, utilities
│   ├── Dockerfile
│   ├── nginx.conf               # prod: static only (Caddy in front)
│   └── nginx.local.conf         # local compose: static + /api + /s3
├── ml/
│   ├── simulator/               # synthetic deal-funnel generator (Go)
│   ├── notebooks/               # LightGBM training (uv, Python)
│   ├── reports/                 # AUC, calibration, SHAP reports
│   ├── features.json            # feature manifest (shared with Go extractor)
│   └── golden/                  # golden test rows for the Go inference
├── loadtest/k6/                 # load testing scenario
├── .github/workflows/
│   ├── ci.yml                   # mandatory PR checks
│   └── deploy.yml               # tests → E2E → build/push → deploy
├── docker-compose.yml           # local development
├── docker-compose.e2e.yml       # light stack for E2E/K6
├── docker-compose.prod.yml      # production
├── Caddyfile
├── CONTRIBUTING.md
└── Makefile
```

---

## Backend

The layered architecture, error handling, the guide to adding routes, migrations and the test layout live in [CONTRIBUTING.md](CONTRIBUTING.md).

### Embeddings (TEI)

Vectors are produced by [Text Embeddings Inference](https://github.com/huggingface/text-embeddings-inference) with `intfloat/multilingual-e5-small`. In docker-compose it runs as the `tei` service at `http://tei:80` (`TEI_URL`), exposed on `TEI_EXTERNAL_PORT` (default `8090`).

The provider is selected by `EMBEDDING_PROVIDER`: `tei` (real service) or `stub` (default — a deterministic stub client for development without TEI). Other settings: `EMBEDDING_TIMEOUT`, `MAX_INPUT_LENGTH` (input is truncated to this many characters before the model call).

Check TEI health manually:

```bash
docker compose up -d tei
curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8090/health   # expect 200
curl -s -X POST http://localhost:8090/embed -H "Content-Type: application/json" -d '{"inputs":"хочу iPhone"}'
```

With `EMBEDDING_PROVIDER=tei`, `app` starts only after `tei` is healthy (the internal healthcheck polls `/health`).

### API reference

Swagger UI is served by the backend at [`/swagger/index.html`](http://localhost:8080/swagger/index.html); the OpenAPI spec is at `/swagger/doc.json`. Handlers are annotated with [swaggo/swag](https://github.com/swaggo/swag); after changing handlers or DTOs, regenerate the spec:

```bash
# one-time: go install github.com/swaggo/swag/cmd/swag@latest
cd backend && swag init -g cmd/api/main.go -o docs --parseInternal
```

See also: [CONTRIBUTING.md](CONTRIBUTING.md) · [Swagger UI](http://localhost:8080/swagger/index.html).

### Realtime (WebSocket)

Live chain/request/item updates are pushed over WebSocket (`/api/v1/ws`). Authentication uses
a short-lived one-time ticket (`/api/v1/ws/ticket`, sent as a WebSocket subprotocol, never in a
query string). Allowed origins are set via `WS_ALLOWED_ORIGINS` (comma-separated, fail-closed);
WS_* settings are validated at startup — a non-positive duration or a negative limit is a
startup error. Implementation is the `internal/realtime` package (hub, tickets, emitter).

Known limitation: the hub and the one-time ticket store are in-process, so the backend
runs as a **single replica**. Horizontal scaling requires either sticky sessions for
`/ws*` or moving fan-out (and the ticket store) to Postgres `LISTEN/NOTIFY` / Redis PubSub.

---

## Frontend

### Environment

| Variable | Default | Purpose |
|----------|---------|---------|
| `VITE_API_BASE_URL` | (empty, relative `/api`) | Base API URL (the `/api/v1` prefix is added automatically). Empty = through the Vite proxy to `:8080` |

A request with an empty `VITE_API_BASE_URL` goes relative through the Vite proxy (`vite.config.ts`, `server.proxy`) to `http://localhost:8080`, so the backend needs no CORS. Photos arrive as `imageUrl` from `S3_PUBLIC_ENDPOINT` and are used as-is.

### Feature-Sliced Design

Layers (top to bottom, imports only downward):

- `src/app` — providers, routing, global styles, store.
- `src/pages` — screens.
- `src/features` — user scenarios.
- `src/entities` — business entities.
- `src/shared` — reusable UI components, API client, utils, types.

### Scripts

```bash
npm run lint          # ESLint (incl. FSD import rules)
npm run lint:fix
npm run format        # Prettier
npm run format:check
npm run test          # Vitest
npm run typecheck     # tsc --noEmit
```

### Realtime (WebSocket)

The client connects to `/api/v1/ws` via the native WebSocket API; the code lives in
`shared/realtime` (client, `RealtimeProvider`, TanStack Query cache invalidation). The feature
is gated by the `VITE_REALTIME_ENABLED` build arg: with `false` the app falls back to polling
and returns to it automatically while the socket is down (e.g. a rejected handshake).

---

## Team

Developed during the Avito hackathon "Code Laboratory".

Backend: [aabbuukkaarr8](https://github.com/aabbuukkaarr8), [Linempy](https://github.com/Linempy), [Danya-programmer](https://github.com/Danya-programmer).