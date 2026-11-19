# Portly

Portly is a lightweight tunneling and webhook forwarding service. It exposes a local HTTP server to the internet through a relay server and forwards incoming webhook requests to it, similar to ngrok or a self-hosted webhook relay.

## How it works

```
Webhook / HTTP request
        │
        ▼
  Relay Server  ──(WebSocket)──▶  Portly CLI  ──(HTTP POST)──▶  Local server
        ▲                                                          │
        └──────────────── response body ───────────────────────────┘
```

1. The CLI authenticates against the API server (`node-backend`) and stores your API key in the OS keyring.
2. `portly forward <local-url>` opens a WebSocket connection to the relay server, identified by your API key.
3. Requests received by the relay server are pushed to the CLI over the WebSocket.
4. The CLI replays each request as an HTTP `POST` to your local URL and sends the response back through the relay to the original caller.
5. The relay stores a record of every delivery (request, response status, duration, errors) in PostgreSQL; you can list it through the API server.

## Repository structure

```
portly/
├── go-cli/          # Go CLI client (login + forward)
├── node-backend/    # Express + Prisma API server (auth, API keys, event history)
├── relay-server/    # Go WebSocket relay + webhook receiver + delivery logging
├── docker-compose.yml
├── test-webhook.js  # Sample local webhook receiver
└── .env.example
```

| Component       | Stack                              | Default port | Purpose                                             |
| --------------- | ---------------------------------- | ------------ | --------------------------------------------------- |
| `node-backend`  | Node.js, Express, Prisma, Postgres | `3000`       | Auth, API keys, webhook event history API           |
| `relay-server`  | Go, Gorilla WebSocket, pgx         | `8000`       | `/ws` clients, `/webhook` ingestion, delivery logs  |
| `go-cli`        | Go, Cobra, go-keyring              | —            | `portly login` and `portly forward` commands        |

See [`go-cli/README.md`](go-cli/README.md) for CLI-specific documentation.

## Prerequisites

- [Docker](https://docs.docker.com/get-docker/) and Docker Compose (for PostgreSQL)
- Go 1.26+
- Node.js 20+ and npm
- A system keyring backend for the CLI (Keychain on macOS, Secret Service on Linux, Credential Manager on Windows)

## Getting started

### 1. Start PostgreSQL

From the repository root:

```bash
cp .env.example .env
# fill in POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DB
docker compose up -d
```

### 2. Run the API server

```bash
cd node-backend
cp .env.example .env
# DATABASE_URL=postgresql://<user>:<password>@localhost:5432/<db>
# JWT_SECRET=<random-secret>
npm install
npx prisma migrate dev
npm run dev
```

The server listens on `http://localhost:3000` by default and exposes:

| Method | Route          | Description                                                     |
| ------ | -------------- | --------------------------------------------------------------- |
| `POST` | `/auth/signup` | Create a user; returns JWT and API key                          |
| `POST` | `/auth/login`  | Authenticate; returns JWT and API key                           |
| `GET`  | `/auth/me`     | Return the authenticated user (requires `Authorization: Bearer`) |
| `GET`  | `/events`      | List your logged webhook deliveries (`?limit=&offset=`)          |
| `GET`  | `/events/:id`  | Get one logged delivery                                           |

Signup body: `{ "email": "...", "password": "...", "slug": "..." }`. Passwords must be at least 8 characters; slugs are 1–40 lowercase letters, digits or hyphens. The `slug` is stored on the user for future subdomain-based routing.

### 3. Run the relay server

```bash
cd relay-server
# create .env with (must point to the same database as node-backend):
# DATABASE_URL=postgresql://<user>:<password>@localhost:5432/<db>
go mod download
go run .
```

The relay validates the `id` API key against PostgreSQL on both endpoints, so it will not start without a reachable database.

The relay listens on `http://localhost:8000`:

| Route      | Description                                                        |
| ---------- | ------------------------------------------------------------------ |
| `/`        | Health check                                                       |
| `/ws`      | WebSocket endpoint for CLI clients (`?id=<apiKey>&localUrl=<url>`) |
| `/webhook` | Incoming webhook endpoint (`?id=<apiKey>`)                         |

### 4. Build and use the CLI

```bash
cd go-cli
cp .env.example .env
# NODE_SERVER_URL=http://localhost:3000
# RELAY_SERVER_URL=http://localhost:8000
# CLI_SERVICE_NAME=portly
# CLI_USER_NAME=<account label>
go build -o portly
./portly login
./portly forward http://localhost:3001/webhook
```

### 5. Send a test webhook

Start the sample receiver in another terminal:

```bash
node test-webhook.js
```

With the CLI forwarding to `http://localhost:3001/webhook`, send a request to the relay:

```bash
curl -X POST "http://localhost:8000/webhook?id=<your-api-key>" \
  -H "Host: demo.localhost:8000" \
  -H "Content-Type: application/json" \
  -d '{"event":"test","message":"hello from portly"}'
```

The `Host` header must contain a dot because the relay splits it to extract the user slug. The request is forwarded to your local server; its status code, headers and body are returned to the original caller. Use the API key from `portly login` as `id`.

### 6. Inspect delivery history

Every authenticated webhook is recorded in the shared PostgreSQL database. Use the JWT from signup/login to list the last 50 deliveries:

```bash
curl -H "Authorization: Bearer <jwt>" "http://localhost:3000/events?limit=50&offset=0"
```

Each entry contains the `requestId`, method, host, slug, path, headers, body, response status code, duration in milliseconds, any error, and a timestamp. `GET /events/:id` returns a single entry. Requests with an invalid API key are rejected before logging and are not stored.

## Configuration

### Root `.env` (Docker Compose)

| Variable            | Description             |
| ------------------- | ----------------------- |
| `POSTGRES_USER`     | PostgreSQL username     |
| `POSTGRES_PASSWORD` | PostgreSQL password     |
| `POSTGRES_DB`       | PostgreSQL database name |

### `node-backend/.env`

| Variable       | Description                          |
| -------------- | ------------------------------------ |
| `DATABASE_URL` | PostgreSQL connection string         |
| `JWT_SECRET`   | Secret used to sign and verify JWTs  |

An optional `PORT` environment variable overrides the default port `3000`.

### `relay-server/.env`

| Variable       | Description                                               |
| -------------- | --------------------------------------------------------- |
| `DATABASE_URL` | PostgreSQL connection string (same database as the API)   |

### `go-cli/.env`

| Variable           | Description                                      |
| ------------------ | ------------------------------------------------ |
| `NODE_SERVER_URL`  | Base URL of the API server (used by `login`)     |
| `RELAY_SERVER_URL` | Base URL of the relay server (used by `forward`) |
| `CLI_SERVICE_NAME` | Keyring service name used to store the API key   |
| `CLI_USER_NAME`    | Keyring account name used to store the API key   |

## Development notes

- The relay server currently routes webhooks by the `id` query parameter (the user's API key) rather than the `Host` header. Host parsing for slug-based routing is scaffolded but not enforced.
- Both relay endpoints validate `id` against the `User.apiKey` column, so the relay shares a database with the API server. WebSocket origins are restricted by Gorilla's default same-origin check.
- The relay rejects request bodies larger than 1 MB with HTTP `413`.
- The CLI replays all forwarded requests as `POST`, passing along the original headers and body.
- The CLI forwards the local server's status code and response headers back to the relay (hop-by-hop headers are stripped), which returns them to the caller.
- A single webhook waits at most 30 seconds for the local server; the CLI uses a 25 second timeout per local request and returns `502` if it fails.
- Concurrent webhooks are supported per client; the CLI processes up to 16 local requests in parallel.
- The API server stores users in PostgreSQL via Prisma; run `npx prisma migrate dev` after schema changes.
- The `add_webhook_events` migration also adds the `User.slug` column, which was missing from the initial migration. Apply it with `npx prisma migrate dev` before starting the relay.
- Delivery logging uses raw SQL from the relay (pgx) against the same `WebhookEvent` table; a failed log write is reported to the relay log but does not affect the forwarded response.
