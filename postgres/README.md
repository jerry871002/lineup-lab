# Postgres

This directory contains the local Postgres image and schema bootstrap files for
Lineup Lab.

## Current Local Bootstrap

- [`init-table.sql`](./init-table.sql)
  is copied into `/docker-entrypoint-initdb.d/`
- the file is applied only when Postgres starts with a brand-new empty data
  directory
- this still makes fresh local development easy to boot today
- however, it is now transitional bootstrap convenience rather than the
  long-term schema source of truth

## Synthetic Demo Roster

A fresh Postgres data directory also runs [`seed-demo.sql`](./seed-demo.sql)
after `init-table.sql`. It inserts 12 fictional players for
**Demo Comets (Synthetic), 2024** so the roster and simulator work immediately
without scraping a third-party website. These are synthetic sample statistics,
not real player data. They have distinct names, valid batting counts, and
non-zero out probabilities.

To import the same roster into an existing local database, run from the repo root
with the Compose stack running:

```sh
docker compose exec -T postgres sh -c \
  'psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$POSTGRES_DB"' \
  < postgres/seed-demo.sql
```

The import is safe to repeat: `ON CONFLICT (name, team, year) DO NOTHING`
preserves existing team data and any edits to demo players. Rebuilding the image
alone does not import data into an existing volume because Postgres init scripts
only run for a new data directory. Use the import command above; no volume reset
is required.

After starting or importing, select **Demo Comets (Synthetic)** and **2024** in
the app. The seed contains more than nine players so users can compare lineups
and try optimization.

CI runs `bash postgres/tests/demo_seed.sh` against PostgreSQL. The test creates
and removes its own test database, checks fresh imports and repeated imports,
validates all 12 batters, and verifies existing teams and edited demo rows survive.
It accepts the standard `PGHOST`, `PGPORT`, `PGUSER`, `PGPASSWORD`, and `PGDATABASE`
connection variables; the test user needs permission to create databases.

## Migration Strategy

The project now treats Atlas-managed migrations as the source of truth for
schema evolution.

Per [ADR 0003](../docs/adr/0003-shared-postgres-migration-strategy.md):

- the ordered migration chain under `postgres/migrations/` is the source of
  truth
- fresh databases and upgraded databases should both converge by applying that
  migration chain
- Atlas versioned migrations are the primary workflow for this repo
- schema changes should be reviewed with explicit service ownership, such as
  `auth`, `stats`, or a future scoreboard service
- migrations should be applied by a dedicated migration step, not by app
  startup side effects
- CI should verify both fresh-install and upgrade-path behavior
- `init-table.sql` can remain temporarily during the transition, but it should
  not diverge from the migration chain and should not be treated as the
  canonical schema forever

## What To Do For Future Schema Changes

1. Add a forward-only SQL migration under `postgres/migrations/`
   - let Atlas create the ordered version prefix
   - use a descriptive suffix with service ownership, for example
     `20260412210000_auth_add_users_display_name.sql`
   - include a top-of-file ownership comment such as `-- owner: auth`
2. Mention schema ownership and migration impact in the PR
3. Keep `init-table.sql` aligned only while the transition period still exists

Do not rely on service startup to apply schema changes. The intended future
flow is:

1. start or reach the target database
2. run the Atlas migration step once
3. deploy the service versions that depend on the new schema

The Atlas integration itself is a follow-up implementation step. This issue
chooses the workflow and review expectations.
