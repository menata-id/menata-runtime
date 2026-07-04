# Development Guide

> This document describes how to set up and run the Menata Runtime Prototype locally.

---

## Prerequisites

| Tool | Version | Purpose |
|------|---------|---------|
| Go | 1.22+ | Runtime engine |
| PostgreSQL | 14+ | Runtime Metadata + Business Data storage |
| Node.js | 18+ | Tailwind CSS build |
| templ | latest | Type-safe HTML templates |

---

## Installation

### 1. Clone the repository

```bash
git clone https://github.com/menata-id/menata.git
cd menata/runtime/prototype
```

### 2. Install Go dependencies

```bash
go mod tidy
```

### 3. Install templ

```bash
go install github.com/a-h/templ/cmd/templ@latest
```

### 4. Install Node dependencies

```bash
npm install
```

### 5. Set up PostgreSQL

Create a database:

```sql
CREATE DATABASE menata_prototype;
```

### 6. Configure environment

Copy the example environment file:

```bash
cp .env.example .env
```

Edit `.env`:

```env
DATABASE_URL=postgres://postgres:password@localhost:5432/menata_prototype?sslmode=disable
PORT=3100
```

### 7. Run database migrations

```bash
make migrate-up
```

### 8. Seed example Runtime Metadata

```bash
make seed
```

---

## Running the Prototype

### Build CSS

```bash
npm run build:css
```

### Generate Templ files

```bash
templ generate
```

### Build and run

```bash
make build
make run
```

Or in development mode with live reload:

```bash
make dev
```

The application will be available at `http://localhost:3100`.

---

## Project Structure

```
runtime/prototype/
├── cmd/
│   └── server/
│       └── main.go          ← entry point
├── internal/
│   ├── interpreter/         ← Runtime Metadata → Application Model
│   ├── router/              ← HTTP routing from metadata
│   ├── renderer/            ← Templ-based HTML rendering
│   ├── executor/            ← Event execution
│   ├── constraint/          ← Constraint enforcement
│   ├── permission/          ← Permission enforcement
│   ├── metadata/            ← Runtime Metadata loading + validation
│   └── model/               ← Application Model (in-memory)
├── web/
│   ├── templates/           ← Templ templates
│   └── static/
│       └── css/             ← Tailwind output
├── migrations/              ← Database migrations
├── seeds/                   ← Example Runtime Metadata seeds
├── docs/                    ← Documentation
├── .env.example
├── Makefile
├── go.mod
└── go.sum
```

---

## Makefile Commands

| Command | Description |
|---------|-------------|
| `make build` | Build the server binary |
| `make run` | Run the server |
| `make dev` | Run with live reload |
| `make migrate-up` | Apply database migrations |
| `make migrate-down` | Rollback last migration |
| `make seed` | Load example Runtime Metadata |
| `make generate` | Run templ generate |
| `make build:css` | Build Tailwind CSS |

---

## Adding a New Machine

1. Define Business Knowledge using Menata Language (see `docs/examples/`)
2. Create Runtime Metadata (YAML) describing the machine
3. Load the metadata via `make seed` or the admin interface
4. The runtime automatically realizes the new machine as a running application

No Go code changes are required to add a new machine.

---

## Troubleshooting

**Runtime Metadata fails to load**

Check validation errors in the server log.

Invalid Runtime Metadata is rejected.

The server will report the specific validation failure.

**CSS changes not reflected**

Run `npm run build:css` to rebuild Tailwind output.

**Templ changes not reflected**

Run `templ generate` to regenerate Go template files.
