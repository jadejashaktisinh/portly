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

	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
	"github.com/zalando/go-keyring"
)

type message struct {
	Id       string
	Body     string
	LocalUrl string
	Header   map[string][]string
}

// forwardCmd represents the forward command
var forwardCmd = &cobra.Command{
	Use:   "forward",
	Short: "forward command is used to forward requests from the relay server to the local server",
	Long:  `forward command is used to forward requests from the relay server to the local server. It connects to the relay server using a websocket connection, listens for incoming messages, and forwards them to the local server. It also sends the response back to the relay server.`,

	Run: func(cmd *cobra.Command, args []string) {

		localUrl := args[0]
		forward(localUrl)

	},
}

func init() {
	portlyCmd.AddCommand(forwardCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// forwardCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// forwardCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func forward(localUrl string) {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	service := os.Getenv("CLI_SERVICE_NAME")
	user := os.Getenv("CLI_USER_NAME")
	secret, err := keyring.Get(service, user)
	if err != nil {
		log.Fatalf("Failed to get secret: %v", err)
	}
	fmt.Printf("Retrieved secret: %s\n", secret)

	fmt.Printf("Local URL: %s\n", localUrl)
	baseStr := os.Getenv("RELAY_SERVER_URL")
	u, err := url.Parse(baseStr)
	if err != nil {
		log.Fatal("Invalid RELAY_SERVER_URL base path:", err)
	}

	// 2. Set the exact path segment for the websocket endpoint
	u.Path = "/ws"
	params := url.Values{}
	params.Add("id", secret)
	params.Add("localUrl", localUrl)
	u.RawQuery = params.Encode()

	// 4. Convert back to string format
	relayServerURL := u.String()
	fmt.Printf("Relay Server URL: %s\n", relayServerURL)
	conn, _, err := websocket.DefaultDialer.Dial(relayServerURL, nil)
	if err != nil {
		log.Fatal("connect error:", err)
	}
	defer conn.Close()

	fmt.Println("connected to", relayServerURL)
	fmt.Println("type a message and press Enter (Ctrl+C to quit)")

	// receive messages in background
	go listener(conn)

}

func listener(conn *websocket.Conn) {
	for {
		msg := message{}
		err := conn.ReadJSON(&msg)
		if err != nil {
			log.Println("read error:", err)
			return
		}
		fmt.Printf("Received message: id=%s, body=%s, localUrl=%s, header=%v\n", msg.Id, msg.Body, msg.LocalUrl, msg.Header)
		resp, err := SendRequestToLocalServer(msg.LocalUrl, []byte(msg.Body), msg.Header)
		if err != nil {
			log.Println("error sending request to local server:", err)
			continue
		}
		fmt.Printf("Response from local server: %s\n", resp)
		// send response back to relay server
		responseMsg := message{Id: msg.Id, Body: resp, LocalUrl: msg.LocalUrl, Header: msg.Header}
		err = conn.WriteJSON(responseMsg)
		if err != nil {
			log.Println("write error:", err)
			return
		}
		log.Println("sent response back to relay server")

	}
}

func SendRequestToLocalServer(localUrl string, body []byte, header map[string][]string) (string, error) {
	httpClient := &http.Client{}
	req, err := http.NewRequest("POST", localUrl, bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}
	for key, values := range header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(respBody), nil
}
