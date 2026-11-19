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

- Go 1.26+
- A running Portly relay server and API server
- A system keyring backend (Keychain on macOS, Secret Service on Linux, Credential Manager on Windows)

## Build

```bash
git clone <repository-url>
cd go-cli
go mod download
go build -o portly
```

## Configuration

The CLI reads configuration from a `.env` file in the working directory. Copy the example file and fill in the values:

```bash
cp .env.example .env
```

| Variable           | Description                                        |
| ------------------ | -------------------------------------------------- |
| `NODE_SERVER_URL`  | Base URL of the API server (used by `login`)       |
| `RELAY_SERVER_URL` | Base URL of the relay server (used by `forward`)   |
| `CLI_SERVICE_NAME` | Keyring service name used to store the API key     |
| `CLI_USER_NAME`    | Keyring account name used to store the API key     |

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

The CLI connects to `RELAY_SERVER_URL/ws` and forwards every incoming request to the given local URL. Press `Ctrl+C` to stop forwarding.

### Help

```bash
./portly --help
./portly login --help
./portly forward --help
```

## Project structure

```
go-cli/
├── main.go          # Entry point
├── cmd/
│   ├── root.go      # Root Cobra command
│   ├── login.go     # `login` command: authentication + keyring storage
│   └── forward.go   # `forward` command: WebSocket listener and local request forwarding
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
