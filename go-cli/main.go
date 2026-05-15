package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"

	"github.com/gorilla/websocket"
)

type message struct {
	Id       string
	Body     string
	LocalUrl string
	Header   map[string][]string
}

func main() {
	if len(os.Args) <= 2 {
		log.Fatal("usage: go-cli <client-id> [local-url]")
	}
	id := os.Args[1]
	localUrl := os.Args[2]
	serverURL := "ws://localhost:8000/ws?id=" + id + "&localUrl=" + localUrl

	conn, _, err := websocket.DefaultDialer.Dial(serverURL, nil)
	if err != nil {
		log.Fatal("connect error:", err)
	}
	defer conn.Close()

	fmt.Println("connected to", serverURL)
	fmt.Println("type a message and press Enter (Ctrl+C to quit)")

	// receive messages in background
	go func() {
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
			responseMsg := message{Id: id, Body: resp, LocalUrl: localUrl}
			err = conn.WriteJSON(responseMsg)
			if err != nil {
				log.Println("write error:", err)
				return
			}
			log.Println("sent response back to relay server")

		}
	}()

	// send messages from stdin
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	scanner := bufio.NewScanner(os.Stdin)
	inputCh := make(chan string)
	go func() {
		for scanner.Scan() {
			inputCh <- scanner.Text()
		}
	}()

	for {
		select {
		case line := <-inputCh:
			if err := conn.WriteMessage(websocket.TextMessage, []byte(line)); err != nil {
				log.Fatal("send error:", err)
			}
		case <-interrupt:
			conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			return
		}
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
