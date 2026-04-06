# Lineup Lab

This directory contains the React frontend for Lineup Lab.

## Features

- Drag-and-drop interface for managing lineups
- Separate sections for lineup and roster
- Real-time updates to the lineup as players are moved
- Simple and intuitive user interface
- Simulate the performance of the created lineup

## Usage

Run the full local stack from the repository root:

```sh
docker compose up --build
```

Then open [http://localhost:8080](http://localhost:8080).

The browser-facing app is served through the `gateway` service, which also proxies `/api/*` to the internal Go services.

To run the frontend tests from this directory, use:

```sh
npm test
```

To validate the production bundle from this directory, use:

```sh
npm run build
```

For the full local architecture, Docker Compose workflow, and environment variable setup, use the repository root README.
