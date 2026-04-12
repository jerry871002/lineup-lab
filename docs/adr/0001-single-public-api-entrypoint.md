# ADR 0001: single public API entrypoint through the gateway

## Status

Accepted

## Context

Lineup Lab has multiple backend services with different responsibilities:

- `stats` for roster and batting data
- `simulation` for simulation and optimization
- `auth` for user and session concerns

Originally, the frontend talked directly to multiple backend services. That
made the browser aware of internal service boundaries and created unnecessary
coupling between the UI and backend topology.

As the project grows to include authentication, sessions, scoreboards, and
Kubernetes deployment, that shape becomes harder to secure and harder to
operate cleanly.

## Decision

The browser will talk to a single public entrypoint: the `gateway` service.

The gateway is responsible for:

- serving the frontend application
- exposing the single public `/api` surface
- routing requests to internal backend services by path

Current routing includes:

- `/api/teams` and `/api/batting` to `stats`
- `/api/simulate` and `/api/optimize` to `simulation`

Future auth and user routes will also be exposed through the same public origin.

## Alternatives Considered

### Let the frontend call each backend service directly

Rejected.

That approach is simple at first, but it leaks internal topology to the
browser, complicates CORS, and makes auth/session rollout harder because
multiple services become browser-facing.

### Make auth the only browser-facing service immediately

Deferred.

That may become a good long-term backend-for-frontend design, but a gateway is
the simpler first step. It gives the project one public origin without forcing
all orchestration into `auth` right away.

### Keep direct browser access in local development and only unify traffic in Kubernetes

Rejected.

That would create two different mental models: one for local work and one for
deployment. Using the gateway locally keeps the architecture consistent.

## Consequences

- the frontend should use relative `/api/...` paths instead of direct service URLs
- backend services remain internal even if they are still exposed on localhost
  temporarily for debugging
- auth, cookies, CSRF, rate limiting, and observability can be designed around
  one browser-facing origin
- the local Docker Compose architecture maps more naturally to a future
  Kubernetes ingress or gateway
