# ADR 0002: auth owns users and sessions

## Status

Accepted

## Context

Lineup Lab now has multiple backend services:

- `stats` for roster and batting data
- `simulation` for lineup simulation and optimization
- `auth` for user and session concerns behind the gateway

The project currently uses one shared Postgres instance for local development.
That shared database now contains auth-domain tables:

- `users`
- `sessions`

Without an explicit ownership rule, future work can blur service boundaries.
For example, `stats` or `simulation` could start writing auth data
directly because the tables are physically nearby. That would make the system
harder to reason about and harder to secure.

## Decision

`auth` is the sole owner of the `users` and `sessions` tables.

Ownership means:

- `auth` defines the application behavior for creating, updating, revoking,
  and reading auth-domain records
- only `auth` may perform direct writes to `users` and `sessions`
- other services must not update those tables directly
- browser-facing auth routes are exposed through `auth` behind the gateway

For now, the tables may remain in the shared Postgres instance used by local
development, but logical ownership still belongs to `auth`.

## Alternatives Considered

### Let stats own auth tables

Rejected.

`stats` is a read-oriented stats service. Making it responsible for
identity and session state would mix unrelated concerns and make its boundary
less clear.

### Let simulation own auth tables

Rejected.

`simulation` is a compute-oriented service. It should stay focused on
simulation behavior, not identity or browser session management.

### Allow any service to read and write auth tables directly

Rejected.

That would be convenient in the short term, but it would weaken ownership,
increase coupling, and make later authorization and auditing work messier.

### Give auth its own separate database immediately

Deferred.

That may become a good long-term direction, but it adds operational complexity
before this project needs it. A shared Postgres instance with explicit logical
ownership is a good intermediate step.

## Consequences

- future auth features such as registration, login, logout, and session
  validation should be implemented in `auth`
- session cookies and protected user flows should be designed around
  `auth` as the browser-facing auth service
- future schema and migration work should preserve `auth` ownership of
  `users` and `sessions`
- other services should consume auth state through APIs or derived data, not by
  writing auth tables directly
