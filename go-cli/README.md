# Portly CLI

A command-line client for **Portly**, a lightweight tunneling service that exposes your local HTTP server to the internet and forwards incoming webhook requests to it.

## How it works

```
Webhook / HTTP request
        │
        ▼
  Relay Server  ──(WebSocket)──▶  Portly CLI  ──(HTTP POST)──▶  Local server
        ▲                                                          │
        └──────────────── response body ───────────────────────────┘
```

1. `portly login` authenticates against the API and stores your API key in the operating system's keyring.
2. `portly forward <local-url>` opens a WebSocket connection to the relay server.
3. Each request the relay receives is pushed to the CLI over the WebSocket; the CLI replays it as an HTTP `POST` to your local URL and sends the status code, response headers and body back through the relay.

## Prerequisites

- A system keyring backend (Keychain on macOS, Secret Service/GNOME Keyring on Linux, Credential Manager on Windows)
- Either a downloaded binary or Go 1.26+ to build from source

## Install

Download the binary for your OS from the [GitHub Releases](../../releases) page, rename it to `portly` and put it on your `PATH`. Tagged `v*` releases are built automatically for Linux, macOS and Windows.

To build from source:

```bash
git clone <repository-url>
cd go-cli
go mod download
go build -o portly
```

## Configuration

The CLI works out of the box against the hosted service. Configuration precedence is:

**flags > environment variables > `.env` file > built-in defaults**

| Flag                | Environment variables                     | Default                            |
| ------------------- | ----------------------------------------- | ---------------------------------- |
| `--api-url`         | `PORTLY_API_URL`, `NODE_SERVER_URL`       | `https://portly-wscl.onrender.com` |
| `--relay-url`       | `PORTLY_RELAY_URL`, `RELAY_SERVER_URL`    | `wss://portly-wscl.onrender.com`   |
| `--keyring-service` | `PORTLY_KEYRING_SERVICE`, `CLI_SERVICE_NAME` | `portly`                        |
| `--keyring-user`    | `PORTLY_KEYRING_USER`, `CLI_USER_NAME`    | current OS username                |

A `.env` file is optional. For local development, copy the example and fill in the values:

```bash
cp .env.example .env
```

To point at a local stack: `./portly forward --api-url http://localhost:3000 --relay-url ws://localhost:8000 http://localhost:3001/webhook`.

## Usage

### Authenticate

```bash
./portly login
```

You will be prompted for your email and password. On success the API key is saved to the system keyring so you don't need to log in again.

### Forward traffic

```bash
./portly forward http://localhost:3000
```

The CLI connects to the relay's `/ws` endpoint and forwards every incoming request to the given local URL. It sends periodic pings to keep the connection alive and reconnects with backoff if the relay restarts. Press `Ctrl+C` to stop forwarding. Each delivery is recorded by the relay and can be inspected later via `GET /events` on the API server.

### Help

```bash
./portly --help
./portly --version
./portly login --help
./portly forward --help
```

## Project structure

```
go-cli/
├── main.go          # Entry point
├── cmd/
│   ├── root.go      # Root Cobra command and flags
│   ├── config.go    # Config precedence and defaults
│   ├── login.go     # `login` command: authentication + keyring storage
│   └── forward.go   # `forward` command: WebSocket listener, reconnect and forwarding
├── .env.example     # Environment variable template
└── go.mod
```

## Dependencies

- [Cobra](https://github.com/spf13/cobra) — CLI framework
- [Gorilla WebSocket](https://github.com/gorilla/websocket) — relay connection
- [godotenv](https://github.com/joho/godotenv) — `.env` loading
- [go-keyring](https://github.com/zalando/go-keyring) — secure API key storage

## License

No license has been specified for this project yet.
