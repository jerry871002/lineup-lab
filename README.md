# Lineup Lab

Lineup Lab is a microservice-style baseball lineup simulator built to practice Go backend development, React frontend work, Docker-based local environments, and Kubernetes-oriented operational patterns.

## Local Architecture

The local stack has five services:

- `frontend`
  - React app served on `http://localhost:8080`
- `stat-api-server`
  - Go API for roster and batting data on `http://localhost:8082`
- `game-simulation`
  - Go API for lineup simulation and optimization on `http://localhost:8081`
- `db`
  - Postgres database on `localhost:5432`
- `pgadmin`
  - Database admin UI on `http://localhost:8083`

Request flow:

1. The frontend requests teams and batting stats from `stat-api-server`
2. The frontend submits a lineup to `game-simulation`
3. `stat-api-server` reads batting data from Postgres

## Local Development

### 1. Create a local env file

Copy the example env file:

```sh
cp .env.example .env
```

The default `.env.example` values are set up for local development, but you should still update placeholder credentials such as:

- `POSTGRES_PASSWORD`
- `PGADMIN_DEFAULT_PASSWORD`
- `STAT_API_SERVER_DATABASE_URL`

If you change the database credentials, keep `STAT_API_SERVER_DATABASE_URL` in sync with them.

### 2. Start the stack with Docker Compose

```sh
docker compose up --build
```

This starts the full local environment with the frontend, both Go services, Postgres, and pgAdmin.

### 3. Open the app

- frontend: [http://localhost:8080](http://localhost:8080)
- stat API: [http://localhost:8082/teams](http://localhost:8082/teams)
- simulation API: [http://localhost:8081/healthz](http://localhost:8081/healthz)
- pgAdmin: [http://localhost:8083](http://localhost:8083)

## Frontend-Only Development

If you want to run the frontend dev server directly:

```sh
cd frontend
npm start
```

The frontend development server uses `frontend/.env.development` to target the local backend services by default:

- `VITE_STAT_API_BASE_URL=http://localhost:8082`
- `VITE_SIMULATION_API_BASE_URL=http://localhost:8081`

Open [http://localhost:3000](http://localhost:3000) to use the Vite development server.

## Configuration

Key local environment variables:

### Database

- `POSTGRES_DB`
- `POSTGRES_USER`
- `POSTGRES_PASSWORD`
- `STAT_API_SERVER_DATABASE_URL`

### Service ports

- `STAT_API_SERVER_PORT`
- `STAT_API_SERVER_HOST_PORT`
- `GAME_SIMULATION_PORT`
- `GAME_SIMULATION_HOST_PORT`

### Service behavior

- `STAT_API_SERVER_ALLOWED_ORIGIN`
- `GAME_SIMULATION_ALLOWED_ORIGIN`
- `GAME_SIMULATION_DEBUG`

### Database admin

- `PGADMIN_DEFAULT_EMAIL`
- `PGADMIN_DEFAULT_PASSWORD`

Notes:

- `docker-compose.yml` requires several env vars explicitly and fails fast if they are missing
- the frontend container build uses service host ports to derive its API base URLs unless you override them explicitly

## Health And Readiness Endpoints

Both Go services expose Kubernetes-friendly probe endpoints:

### `game-simulation`

- `GET /healthz`
- `GET /readyz`

Both return `200 OK` when the process is available.

### `stat-api-server`

- `GET /healthz`
- `GET /readyz`

`/readyz` is backed by a database ping, so it returns success only when the API can reach Postgres.

## Validation And Safety Checks

The simulator now performs stricter input validation:

- rejects malformed JSON
- rejects unknown JSON fields
- rejects duplicate batter names
- rejects invalid stat combinations such as `hit > at_bat`

This protects the service from obviously invalid simulation input and makes the API behavior easier to reason about.

## CI

Current GitHub Actions coverage includes:

- Compose configuration validation
  - verifies default and overridden local port/env wiring
  - verifies required env vars fail fast
- Go static checks
  - `gofmt`
  - `go vet`
  - `golangci-lint`
- Go service build/test workflows
  - `game-simulation`
  - `stat-api-server`

## Repository Pointers

- frontend app: [frontend/README.md](frontend/README.md)
- local env template: [.env.example](.env.example)
- contributor workflow: [CONTRIBUTING.md](CONTRIBUTING.md)
