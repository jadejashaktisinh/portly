package main

import (
	"fmt"
	"log"
	"net/http"

	"jadejashaktisinh/relay-server/db"
	"jadejashaktisinh/relay-server/socket"
	"jadejashaktisinh/relay-server/webhook"
)

func main() {
	db.Connect()
	defer db.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Hello, World!")
	})
	mux.HandleFunc("/ws", socket.HandleWebSocket)
	mux.HandleFunc("/webhook", webhook.HandleWebhook)

	fmt.Println("Starting server on :8000")
	if err := http.ListenAndServe(":8000", mux); err != nil {
		log.Fatal(err)
	}
}
