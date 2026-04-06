# Auth API

This service is the planned FastAPI-based home for user and session concerns.

Current scope:
- health and readiness endpoints
- database configuration and connectivity checks
- SQLAlchemy models for the `users` and `sessions` tables
- placeholder auth and user routes that reserve the public API shape

Expected public routes:
- `POST /auth/register`
- `POST /auth/login`
- `POST /auth/logout`
- `GET /users/me`

Run locally once dependencies are installed:

```sh
uvicorn app.main:app --app-dir auth-api --reload
```
