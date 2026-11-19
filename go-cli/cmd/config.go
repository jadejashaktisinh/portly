package cmd

import (
	"errors"
	"os"
	"os/user"
	"time"

	"github.com/joho/godotenv"
)

const (
	defaultAPIServerURL   = "https://portly-wscl.onrender.com"
	defaultRelayServerURL = "wss://portly-wscl.onrender.com"
	defaultKeyringService = "portly"
	relayPingInterval     = 60 * time.Second
)

var (
	flagAPIURL         string
	flagRelayURL       string
	flagKeyringService string
	flagKeyringUser    string
)

func loadDotEnv() {
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		os.Stderr.WriteString("warning: could not load .env: " + err.Error() + "\n")
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func apiServerURL() string {
	value := firstNonEmpty(
		flagAPIURL,
		os.Getenv("PORTLY_API_URL"),
		os.Getenv("NODE_SERVER_URL"),
	)
	if value == "" {
		return defaultAPIServerURL
	}
	return value
}

func relayServerURL() string {
	value := firstNonEmpty(
		flagRelayURL,
		os.Getenv("PORTLY_RELAY_URL"),
		os.Getenv("RELAY_SERVER_URL"),
	)
	if value == "" {
		return defaultRelayServerURL
	}
	return value
}

func keyringService() string {
	value := firstNonEmpty(
		flagKeyringService,
		os.Getenv("PORTLY_KEYRING_SERVICE"),
		os.Getenv("CLI_SERVICE_NAME"),
	)
	if value == "" {
		return defaultKeyringService
	}
	return value
}

func keyringUser() string {
	value := firstNonEmpty(
		flagKeyringUser,
		os.Getenv("PORTLY_KEYRING_USER"),
		os.Getenv("CLI_USER_NAME"),
	)
	if value != "" {
		return value
	}
	if current, err := user.Current(); err == nil && current.Username != "" {
		return current.Username
	}
	return "portly"
}
