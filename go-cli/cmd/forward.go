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
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
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
	Use:   "forward",
	Short: "forward command is used to forward requests from the relay server to the local server",
	Long:  `forward command is used to forward requests from the relay server to the local server. It connects to the relay server using a websocket connection, listens for incoming messages, and forwards them to the local server. It also sends the response back to the relay server.`,

	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			fmt.Println("Provide local url to forward")
			return
		}
		forward(args[0])
	},
}

func init() {
	rootCmd.AddCommand(forwardCmd)
}

func forward(localUrl string) {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}
	service := os.Getenv("CLI_SERVICE_NAME")
	user := os.Getenv("CLI_USER_NAME")
	secret, err := keyring.Get(service, user)
	if err != nil {
		log.Fatalf("Failed to get API key: %v", err)
	}

	fmt.Printf("Local URL: %s\n", localUrl)
	baseStr := os.Getenv("RELAY_SERVER_URL")
	u, err := url.Parse(baseStr)
	if err != nil {
		log.Fatal("Invalid RELAY_SERVER_URL base path:", err)
	}

	u.Path = "/ws"
	params := url.Values{}
	params.Add("id", secret)
	params.Add("localUrl", localUrl)
	u.RawQuery = params.Encode()

	relayServerURL := u.String()
	fmt.Printf("Relay server: %s://%s%s\n", u.Scheme, u.Host, u.Path)
	conn, _, err := websocket.DefaultDialer.Dial(relayServerURL, nil)
	if err != nil {
		log.Fatal("connect error:", err)
	}
	defer conn.Close()

	fmt.Println("connected to relay; forwarding requests (Ctrl+C to quit)")
	listener(conn)
}

func listener(conn *websocket.Conn) {
	var writeMu sync.Mutex
	sem := make(chan struct{}, 16)

	for {
		msg := message{}
		if err := conn.ReadJSON(&msg); err != nil {
			log.Println("read error:", err)
			return
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
