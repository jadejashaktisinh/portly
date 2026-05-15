package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/gorilla/websocket"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("usage: go-cli <client-id> [ws-url]")
	}
	id := os.Args[1]
	serverURL := "ws://localhost:8000/ws?id=" + id
	if len(os.Args) > 2 {
		serverURL = os.Args[2] + "?id=" + id
	}

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
			_, msg, err := conn.ReadMessage()
			if err != nil {
				log.Println("disconnected:", err)
				os.Exit(0)
			}
			fmt.Println("server:", string(msg))
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
