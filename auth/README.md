# Auth

This service is the planned FastAPI-based home for user and session concerns.

Current scope:
- health and readiness endpoints
- database configuration and connectivity checks
- SQLAlchemy models for the `users` and `sessions` tables
- Argon2-based password hashing helpers and credential validation
  - configurable hash cost settings for auth environments
  - length-based password validation aligned with NIST/OWASP-style guidance
- placeholder auth and user routes that reserve the public API shape

Ownership:
- `auth` owns the `users` and `sessions` tables
- other services should not write auth-domain tables directly
- the shared Postgres instance does not change that logical ownership model
- auth-owned schema changes should follow the shared migration strategy in [docs/adr/0003-shared-postgres-migration-strategy.md](../docs/adr/0003-shared-postgres-migration-strategy.md)

See the architecture decision record:
- [docs/adr/0002-auth-owns-users-and-sessions.md](../docs/adr/0002-auth-owns-users-and-sessions.md)

Expected public routes:
- `POST /auth/register`
- `POST /auth/login`
- `POST /auth/logout`
- `GET /users/me`

Session and CSRF behavior for those routes is defined in [docs/auth-session-strategy.md](../docs/auth-session-strategy.md).

Run locally once dependencies are installed:

```sh
uvicorn app.main:app --app-dir auth --reload
```
