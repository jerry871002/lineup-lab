# Postgres Migrations

Atlas-managed versioned SQL migrations will live in this directory.

Intended conventions:

- use forward-only SQL files
- let Atlas create the ordered version prefix
- keep filenames descriptive, for example
  `20260412210000_auth_create_sessions.sql`
- add a top-of-file ownership comment such as `-- owner: auth`
- plan for upgrade-path testing as well as fresh-bootstrap testing

Expected future validation:

- fresh-install test: empty database + full migration chain reaches the expected
  schema
- upgrade test: older schema state + pending migrations reaches the same schema

The Atlas wiring is not implemented yet. This directory exists now so the
strategy chosen in
[ADR 0003](../../docs/adr/0003-shared-postgres-migration-strategy.md)
has a concrete home in the repository.
