/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/spf13/cobra"
	"github.com/zalando/go-keyring"
)

const localRequestTimeout = 25 * time.Second

type message struct {
	Id         string
	Body       string
	LocalUrl   string
	Header     map[string][]string
	StatusCode int
	RequestId  string
}

type localResponse struct {
	StatusCode int
	Header     map[string][]string
	Body       string
}

var hopByHopHeaders = map[string]struct{}{
	"Connection":          {},
	"Keep-Alive":          {},
	"Proxy-Authenticate":  {},
	"Proxy-Authorization": {},
	"Te":                  {},
	"Trailer":             {},
	"Transfer-Encoding":   {},
	"Upgrade":             {},
}

var forwardCmd = &cobra.Command{
	Use:   "forward <local-url>",
	Short: "Forward relay requests to a local server",
	Long:  `Connect to the Portly relay over WebSocket, forward every incoming request to a local HTTP server as POST, and send the response back to the caller.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		forward(args[0])
	},
}

func init() {
	rootCmd.AddCommand(forwardCmd)
}

func forward(localURL string) {
	service := keyringService()
	user := keyringUser()
	secret, err := keyring.Get(service, user)
	if err != nil {
		log.Fatalf("Failed to get API key: %v", err)
	}

	relayURL, err := buildRelayURL(secret, localURL)
	if err != nil {
		log.Fatal("Invalid relay URL:", err)
	}
	displayURL, err := url.Parse(relayURL)
	if err != nil {
		log.Fatal("Invalid relay URL:", err)
	}
	displayURL.RawQuery = ""

	fmt.Printf("Local URL: %s\n", localURL)
	fmt.Printf("Relay server: %s\n", displayURL.String())
	connectLoop(relayURL)
}

func buildRelayURL(secret, localURL string) (string, error) {
	u, err := url.Parse(relayServerURL())
	if err != nil {
		return "", err
	}
	u.Path = "/ws"
	params := url.Values{}
	params.Add("id", secret)
	params.Add("localUrl", localURL)
	u.RawQuery = params.Encode()
	return u.String(), nil
}

func connectLoop(relayURL string) {
	backoff := time.Second
	for {
		conn, _, err := websocket.DefaultDialer.Dial(relayURL, nil)
		if err != nil {
			log.Printf("relay connection failed: %v; retrying in %s", err, backoff)
			time.Sleep(backoff)
			if backoff < 30*time.Second {
				backoff *= 2
			}
			continue
		}

		log.Println("connected to relay; forwarding requests (Ctrl+C to quit)")
		err = listen(conn)
		conn.Close()
		if err != nil {
			log.Printf("relay connection lost: %v", err)
		}

		backoff = time.Second
		log.Printf("reconnecting in %s", backoff)
		time.Sleep(backoff)
	}
}

func listen(conn *websocket.Conn) error {
	var writeMu sync.Mutex
	sem := make(chan struct{}, 16)
	done := make(chan struct{})

	ticker := time.NewTicker(relayPingInterval)
	defer ticker.Stop()
	defer close(done)

	go func() {
		for {
			select {
			case <-ticker.C:
				if err := conn.WriteControl(websocket.PingMessage, []byte("keepalive"), time.Now().Add(5*time.Second)); err != nil {
					log.Printf("keepalive ping failed: %v", err)
					return
				}
			case <-done:
				return
			}
		}
	}()

	for {
		msg := message{}
		if err := conn.ReadJSON(&msg); err != nil {
			return err
		}
		log.Printf("received request %s (%d bytes)", msg.RequestId, len(msg.Body))

		sem <- struct{}{}
		go func() {
			defer func() { <-sem }()
			handleRequest(conn, msg, &writeMu)
		}()
	}
}

func handleRequest(conn *websocket.Conn, msg message, writeMu *sync.Mutex) {
	resp, err := SendRequestToLocalServer(msg.LocalUrl, []byte(msg.Body), msg.Header)
	if err != nil {
		log.Printf("error forwarding request %s to local server: %v", msg.RequestId, err)
		resp = localResponse{
			StatusCode: http.StatusBadGateway,
			Body:       "local server error",
		}
	}

	responseMsg := message{
		Id:         msg.Id,
		RequestId:  msg.RequestId,
		StatusCode: resp.StatusCode,
		Header:     resp.Header,
		Body:       resp.Body,
	}

	writeMu.Lock()
	err = conn.WriteJSON(responseMsg)
	writeMu.Unlock()
	if err != nil {
		log.Println("write error:", err)
		return
	}
	log.Printf("sent response for request %s (status %d)", msg.RequestId, resp.StatusCode)
}

func SendRequestToLocalServer(localUrl string, body []byte, header map[string][]string) (localResponse, error) {
	httpClient := &http.Client{Timeout: localRequestTimeout}
	req, err := http.NewRequest(http.MethodPost, localUrl, bytes.NewBuffer(body))
	if err != nil {
		return localResponse{}, err
	}
	for key, values := range header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return localResponse{}, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return localResponse{}, err
	}

	return localResponse{
		StatusCode: resp.StatusCode,
		Header:     responseHeaders(resp.Header),
		Body:       string(respBody),
	}, nil
}

func responseHeaders(h http.Header) map[string][]string {
	out := make(map[string][]string, len(h))
	for key, values := range h {
		if _, skip := hopByHopHeaders[http.CanonicalHeaderKey(key)]; skip {
			continue
		}
		out[key] = append([]string(nil), values...)
	}
	return out
}
