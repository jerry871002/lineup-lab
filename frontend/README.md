# Lineup Lab

This directory contains the React frontend for Lineup Lab.

## Features

- Drag-and-drop interface for managing lineups
- Separate sections for lineup and roster
- Real-time updates to the lineup as players are moved
- Simple and intuitive user interface
- Simulate the performance of the created lineup

## Usage

To start the development server, run:

```sh
npm start
```

The development server loads API URLs from `frontend/.env.development`, so it targets the local backend services by default.

- `VITE_STAT_API_BASE_URL=http://localhost:8082`
- `VITE_SIMULATION_API_BASE_URL=http://localhost:8081`

To run the frontend tests, use:
```sh
npm test
```

Open [http://localhost:3000](http://localhost:3000) to view it in the browser with the Vite development server.

For the full local architecture, Docker Compose workflow, and environment variable setup, use the repository root README.
