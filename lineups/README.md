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

## Personal simulation history

- `POST /lineups/{id}/simulations`: evaluate the authenticated owner's saved
  lineup snapshot and persist the successful result. Send no request body;
  client-provided scores or batting statistics are rejected. Returns 201.
- `GET /simulations`: the owner's latest 100 results, ordered by creation time
  and ID descending. Other owners' results are never included.

Each run returns `id`, `saved_lineup_id`, `lineup_name`, `team`, `year`,
`lineup_revision`, `batter_ids`, `batters`, `average_score`, `average_hits`,
`sample_count`, `simulator_version`, and `created_at`. The snapshot and result
are immutable through this API. Editing or deleting a saved lineup preserves its
history; deletion changes only `saved_lineup_id` to null.

The service sends the stored nine-batter snapshot, without IDs, to the configured
simulator. It requires finite non-negative averages, positive sample count, and
an engine version returned by that server. Failed computations, malformed
metadata, and client-submitted scores do not create history rows. This is personal
history, without ranked rules, certified datasets, or a global scoreboard.

`SIMULATION_TIMEOUT` controls the upstream simulation timeout (default `85s`).
The run request budget adds 5 seconds for history persistence, and the server
write budget adds another 10 seconds to deliver the response. Thus defaults are
85/90/100 seconds without independently configurable, contradictory budgets.
CRUD and history reads retain their 10-second context. The trusted gateway must authenticate the new routes,
verify CSRF for run requests, replace incoming identity headers, and allow enough
upstream time for a run.

## Configuration and migrations

- `DATABASE_URL`: Postgres connection string, required.
- `STATS_BASE_URL`: internal stats origin, required (for example `http://stats:80`).
- `SIMULATION_BASE_URL`: internal simulator origin, required (for example `http://simulation:80`).
- `SIMULATION_TIMEOUT`: positive Go duration for the upstream run (default `85s`, for example `120s`). Invalid or overflowing budgets fail startup. If increased, configure the gateway upstream timeout to at least the resulting write budget (`135s` for `120s`).
- `PORT`: listener port, defaults to 80.

Fresh databases receive the table from `postgres/init-table.sql`. Existing
databases must apply the additive migration before starting the service:

```sh
docker compose exec -T postgres sh -c 'psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$POSTGRES_DB"' < postgres/migrations/20261028190000_lineups_create_saved_lineups.sql
```

Existing databases must also apply the simulation history migration after the
saved-lineups migration and before deploying this service version:

```sh
docker compose exec -T postgres sh -c 'psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$POSTGRES_DB"' < postgres/migrations/20261028230000_lineups_create_simulation_runs.sql
```

Both migrations are repeatable and do not modify existing lineups. It assumes the
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

The service remains private. Gateway routes and the frontend must be deployed
with the corresponding authentication and CSRF enforcement.
