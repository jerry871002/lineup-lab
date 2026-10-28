# Saved lineups service

This internal Go service owns `saved_lineups`. It uses stats' HTTP API to resolve
nine ordered player IDs into a validated batting snapshot. Auth remains the sole
owner of users and sessions; this service never reads or writes auth records.

## API

- `GET /lineups`: the authenticated owner's latest 100 lineups.
- `POST /lineups`: create a lineup.
- `GET /lineups/{id}`: load the owner's lineup and ordered batting snapshot.
- `PUT /lineups/{id}`: update using the last observed `revision`.
- `DELETE /lineups/{id}`: delete the owner's lineup.
- `GET /healthz`, `GET /readyz`: process and database probes.

Create body:

```json
{"name":"My lineup","team":"Demo Comets (Synthetic)","year":2024,"batter_ids":[1,2,3,4,5,6,7,8,9]}
```

Updates use the same fields and `revision` from the previous response. Concurrent
updates return 409 instead of silently overwriting one another. Duplicate names
within an owner's team-season also return 409. Missing and cross-owner resources
both return 404. Owner IDs in bodies or query parameters cannot choose an owner.

The service trusts only `X-Authenticated-User-ID` established by a trusted gateway.
Deploy it on the private backend network with **no host port or public ingress**.
The gateway must authenticate sessions and validate CSRF for every write through
auth's internal verifier, replace incoming identity headers, and strip cookies.
Direct requests with a forged header are not authenticated by this service: its
private network boundary is required, not optional.

## Configuration and migrations

- `DATABASE_URL`: Postgres connection string, required.
- `STATS_BASE_URL`: internal stats origin, required (for example `http://stats:80`).
- `PORT`: listener port, defaults to 80.

Fresh databases receive the table from `postgres/init-table.sql`. Existing
databases must apply the additive migration before starting the service:

```sh
docker compose exec -T postgres sh -c 'psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$POSTGRES_DB"' < postgres/migrations/20261028190000_lineups_create_saved_lineups.sql
```

The migration is repeatable and does not modify existing lineups. It assumes the
existing auth-owned `users` table is present. Atlas wiring remains a separate
repository task.

## Verification

```sh
cd lineups
go test ./...
go vet ./...
LINEUPS_TEST_DATABASE_URL='postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable' go test -race ./...
```

The database test creates and drops only a unique test schema. It verifies fresh
bootstrap plus repeat migration, per-owner CRUD, snapshot/order persistence,
duplicate names, and two concurrent writes using the same revision.

This backend PR intentionally does not expose browser routes or add a frontend.
Gateway enforcement and the saved-lineup UI are follow-ups under #34 and #35.
