# Contributing to Menyayka

This guide covers the backend architecture, conventions and local development workflow. For setup, project overview and the API reference, see the [README](README.md) and the [Swagger UI](http://localhost:8080/swagger/index.html).

## Backend architecture

The backend is layered (layer-first), not feature-first:

```
internal/
  entity/                 # domain entities (structs + domain errors)
  repository/<feature>/   # DB access (postgres), implements contracts from service/<feature>
  service/<feature>/      # business logic; contracts.go — interfaces of what it needs
  handler/<feature>/      # gin HTTP handlers; contracts.go — interfaces the service provides
  infrastructure/         # technical clients without business logic (mailer, codestore, jwt, embeddings)
  realtime/               # WebSocket hub, one-time tickets, event fan-out (backend side of the realtime feature)
  middleware/             # cross-cutting gin middleware (currently Auth / JWT)
  api/                    # common HTTP response/error format (internal/api/response.go)
  core/                   # config, logger, DB pool, router
```

An upper layer depends on a lower one only through an interface from its own `contracts.go`; the implementation is passed explicitly as a constructor argument (manual dependency injection, no DI framework). See `internal/service/user` and `internal/handler/user`.

## Realtime (WebSocket)

`internal/realtime` is the backend side of the realtime feature: an in-memory `Hub`, the WebSocket upgrade at `/api/v1/ws` and one-time ticket issuance at `/api/v1/ws/ticket`. The websocket transport is [`github.com/coder/websocket`](https://github.com/coder/websocket). The client (native WebSocket, gated by the `VITE_REALTIME_ENABLED` build arg) lives in `frontend/src/shared/realtime`.

Conventions:

- The client authenticates with a short-lived one-time ticket sent as a WebSocket subprotocol, never in a query string. Set `WS_ALLOWED_ORIGINS` on the backend (comma-separated, fail-closed).
- When `VITE_REALTIME_ENABLED` is `false` (or while the socket is down) the app falls back to polling.
- The hub and the ticket store are in-process, so the backend runs as a **single replica**. Horizontal scaling needs sticky sessions for `/ws*` or moving fan-out to Postgres `LISTEN/NOTIFY` / Redis PubSub.
- Tests for the package live next to it (e.g. `internal/realtime/hub_test.go`, `internal/realtime/handler_test.go`).

## Error handling

- **Business errors** (`entity/errors.go`) — returned by the service layer (`entity.ErrUserAlreadyExists`, …).
- **DB errors** (`repository/errors.go`) — `repository.MapDBError` maps pgx-specific errors (e.g. a UNIQUE violation) to `repository.ErrDuplicateKey` / `ErrNotFound`; nothing above the repository knows PostgreSQL error codes.
- **HTTP response** (`internal/api/response.go`) — a single format across the project: `{"error": "...", "code": 400}`. The handler maps a business error to an HTTP status (`errors.Is` + `switch`).

## Adding a route

1. Implement a `Handler` with handler methods in `internal/handler/<feature>/` (see `handler/user`).
2. Add the handler explicitly to `router.New(...)` (`internal/core/router/router.go`) and mount its routes on the `api` group (gets the `/api/v1` prefix the frontend expects). For authenticated routes, mount `middleware.Auth(tokenParser)` (see `authProtected` in `router.go`).
3. Build the handler via its `service` and `repository`, and pass it to `router.New(...)` in `cmd/api/main.go`.

## API documentation (Swagger)

The OpenAPI spec is generated from handler annotations with [swaggo/swag](https://github.com/swaggo/swag) and served by `gin-swagger` at `/swagger/index.html` (spec at `/swagger/doc.json`). The generated package lives in `backend/docs` and is committed, so the server always has an up-to-date spec.

When you add or change a handler, DTO or route:

1. Add/update the swag comment block directly above the handler method (`@Summary`, `@Tags`, `@Param`, `@Success`, `@Failure`, `@Security BearerAuth`, `@Router`).
2. Regenerate and rebuild:

```bash
# one-time: go install github.com/swaggo/swag/cmd/swag@latest
cd backend && swag init -g cmd/api/main.go -o docs --parseInternal
```

General API info (title, version, host, `BasePath`, Bearer security definition) lives above `main` in `cmd/api/main.go`. The shared error envelope is `api.ErrorResponse`.

## Migrations

Migrations live in `backend/migrations` as `<number>_<name>.up.sql` / `.down.sql` (golang-migrate format). They are applied by a separate `migrate` service in docker-compose, not by the app itself, so the schema is applied exactly once, before the app starts, with no races and no migration rights in the `app` service.

Apply / roll back manually:

```bash
docker compose run --rm migrate -path=/migrations -database="postgres://menyayka:menyayka@db:5432/menyayka?sslmode=disable" up
docker compose run --rm migrate -path=/migrations -database="postgres://menyayka:menyayka@db:5432/menyayka?sslmode=disable" down 1
```

Add a new migration as a pair of files with the next number, e.g. `000005_add_something.up.sql` / `000005_add_something.down.sql`.

## Tests

Backend tests live separately in `backend/tests/`, mirroring `internal/` (e.g. `internal/service/user` → `tests/service/user`), in an external package (`user_test`) so they test only the public contract.

```bash
cd backend && go test ./...
```

The API E2E suite runs against the lightweight compose stack (see the [README](README.md#getting-started)).
