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
