# api-core

Go HTTP API template: Echo, Postgres (pgx + bob), Redis, goose migrations,
Google OAuth2 sign-in and HS256 access tokens.

## Quick start

```
cp .env.example .env
docker compose up -d                       # postgres + redis
go run ./cmd api --addr 0.0.0.0:3030       # applies pending migrations, then serves
```

`api` always runs `migrate up` before serving. Goose holds a Postgres advisory
lock, so several replicas starting together apply each migration once. Pass
`--skip-migrate` (or `SKIP_MIGRATE=true`) only when migrations are run by a
separate deploy step.

Manual migration commands:
```
go run ./cmd migrate --action up       # apply pending migrations
go run ./cmd migrate --action down     # roll back the last migration
go run ./cmd migrate --action redo     # down + up the last migration
go run ./cmd migrate --action status
go run ./cmd migrate --action version
```

## Endpoints (`/api/v1`)

| Method | Path                    | Auth   | Description                                  |
|--------|-------------------------|--------|----------------------------------------------|
| GET    | `/ping`                 | -      | Liveness                                     |
| GET    | `/auth/google/login`    | -      | Returns `{url}` to Google's consent screen   |
| GET    | `/auth/google/callback` | -      | `?state&code` → `{access_token, user, ...}`  |
| GET    | `/auth/me`              | Bearer | Current user                                 |

Every response uses one envelope: `{"data": ...}` on success,
`{"code": "...", "message": "..."}` on error. Server-side failures (5xx) are
logged and never leak details to clients.

The Google redirect URL usually points at the frontend; the frontend then calls
`/auth/google/callback` with the `state` and `code` it received.

## Architecture

```
cmd/                 CLI: api, migrate
internal/
  config/            env → validated Config (fails fast on bad values)
  container/         DI wiring (samber/do) — the only place that knows concrete types
  domain/            entities (no storage / transport / SDK deps)
  handler/           router + one Module per feature (handler/auth, ...)
  service/<feature>/ business logic; declares the interfaces (ports) it needs
  datastore/         Postgres/Redis implementations of those ports
  migrate/           goose + embedded SQL (internal/migrate/sql)
  bob/               generated models — do not edit
pkg/                 reusable: errorx, jwtx, auth, httpx_echo, arr
```

Dependency direction: `handler → service → (ports) ← datastore`. Each layer
depends on small interfaces owned by the consumer, so implementations can be
swapped (e.g. `jwtx.HMAC` ↔ `auth.CognitoVerifier` behind `httpx.TokenVerifier`)
and services are unit-tested with fakes.

### Adding a feature

1. Migration in `internal/migrate/sql/`, then regenerate models (below).
2. Store in `internal/datastore/<x>store` returning `domain` types; wrap errors
   with `errorx.Wrap(err, errorx.Database)` (no rows → 404, unique → 409).
3. Service in `internal/service/<x>` with a `ports.go` listing what it needs.
4. Handler in `internal/handler/<x>` implementing `handler.Module`
   (`RegisterRoutes(api, requireAuth)`); return errors via `httpx.Fail`.
5. Wire providers in `internal/container` and append the module to `Modules`.

## Code generation

Bob models are written to `internal/bob` (package `bobmodel`). Run against a
migrated database:
```
go run github.com/stephenafamo/bob/gen/bobgen-psql@v0.41.1 -c ./internal/config/bobgen.yaml
```

## Tests and lint

```
go test ./...
TEST_DATABASE_DSN='postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable' go test ./internal/datastore/...
gofumpt -l -w .
golangci-lint run ./...
```
