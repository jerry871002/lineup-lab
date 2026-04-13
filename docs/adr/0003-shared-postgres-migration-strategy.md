# ADR 0003: shared Postgres migration strategy

## Status

Accepted

## Context

Lineup Lab currently initializes Postgres from a single bootstrap SQL file:

- [`postgres/init-table.sql`](../../postgres/init-table.sql)

That approach works for a brand-new local database, but it becomes fragile once
multiple services own schema in the same Postgres instance:

- `stats` owns baseball data tables such as `batting`
- `auth` owns `users` and `sessions`
- future issues will add scoreboard and async-job tables

We need one migration workflow for the shared database rather than one tool per
language or per service. Otherwise we would end up with fragmented ordering,
review, and rollout behavior against the same Postgres instance.

## Decision

Lineup Lab will use Atlas as the repo-wide migration workflow for the shared
Postgres database.

The primary Atlas workflow for this project is versioned migrations.
Declarative or schema-diff features may still be useful as supporting tooling,
but the reviewed and deployed artifact should be an ordered migration chain.

Additional rules:

- the ordered migration chain is the source of truth for schema evolution
- fresh databases and existing databases should both converge by applying the
  migration chain
- migration files live under `postgres/migrations/`
- migration files should stay SQL-first and language-agnostic
- migration filenames should use an Atlas-managed ordered version prefix plus a
  service ownership hint, for example
  `20260412210000_auth_add_users_display_name.sql`
- each migration file must include a top-of-file SQL comment naming the owning
  service, for example `-- owner: auth`
- services must not run schema migrations automatically on application startup
- migrations should run in a dedicated step after the database is reachable and
  before application versions that depend on the new schema are rolled out

`postgres/init-table.sql` may remain temporarily as local bootstrap convenience
while the repo transitions, but it is no longer the long-term source of truth
and should eventually be removed or reduced to setup-only behavior once Atlas is
fully in place.

## Why Atlas

Atlas fits this repo well because:

- it is language-agnostic and works across Go and Python services
- it supports versioned migrations, which matches the immediate need here
- it also has a Kubernetes path later without forcing Kubernetes to be the
  migration entrypoint today
- it avoids making the shared database migration workflow depend on one service
  framework

Versioned migrations are the better primary fit for this repo because they make
upgrade sequencing, rollout ordering, and review more explicit while the project
still uses a shared Postgres database across multiple services.

## Alternatives Considered

### Keep bootstrap SQL only

Rejected.

That remains simple in the short term, but it does not provide a safe upgrade
path for existing environments and makes multi-service schema review harder.

### Use Alembic

Rejected.

Alembic is a Python migration tool commonly paired with SQLAlchemy. It would fit
`auth`, but not the shared repository as a whole. Choosing it would make
database evolution feel owned by the Python stack even when the affected schema
belongs to `stats` or future non-Python services.

### Use separate migration tools per service

Rejected.

That would fragment ordering, CI, and deployment behavior while the database is
still shared. Schema ownership can stay per service, but migration execution
should stay database-wide and unified.

### Use CNCF SchemaHero

Rejected for now.

SchemaHero is the most relevant CNCF-native option and has a strong
Kubernetes-oriented model, but Atlas is a better fit for the current repo stage:
it works cleanly in local development and CI today while still supporting a
future Kubernetes migration story.

### Let each service mutate schema on startup

Rejected.

That increases rollout risk, makes ownership less explicit, and is awkward for
future Kubernetes deployments where schema changes should be applied once in a
controlled step.

## Consequences

- future schema PRs should add Atlas-managed migrations under
  `postgres/migrations/`
- local development, CI, and future Kubernetes deployments should all rely on
  the same migration chain
- fresh-install and upgrade-path testing should both be explicit in CI
- service docs and PRs should call out schema ownership and migration impact
- follow-up implementation work should add:
  - Atlas tooling and commands to the repo
  - migration application in CI
  - a clear path for local and Kubernetes execution

## Testing Expectations

The migration workflow should eventually verify both:

- fresh install:
  - start from an empty database
  - apply the full Atlas migration chain
  - confirm the resulting schema matches expectations
- upgrade path:
  - start from an older schema state
  - apply only pending migrations
  - confirm the resulting schema matches the same expected end state

## Example Future Change

If `auth` later needs to add `display_name` to `users`, the expected flow is:

1. Add a migration such as
   `postgres/migrations/20260412210000_auth_add_users_display_name.sql`
2. Put the schema change in that file, for example:

   ```sql
   -- owner: auth
   ALTER TABLE users
   ADD COLUMN display_name VARCHAR(100);
   ```

3. Apply that migration through the shared Atlas workflow
4. Update the owning service code after the schema contract is clear
5. In CI, verify both:
   - a fresh database migrated from zero
   - an older database upgraded through pending migrations

That keeps fresh installs and upgrade installs converging on the same schema
without maintaining two independent sources of truth.
