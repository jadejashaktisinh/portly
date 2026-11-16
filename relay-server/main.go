package main

import (
	"fmt"
	"jadejashaktisinh/relay-server/db"
	"jadejashaktisinh/relay-server/socket"
	"jadejashaktisinh/relay-server/webhook"
	"net/http"
)

func main() {

	db.Connect()

	fmt.Println("Starting server on :8000")
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, World!")
	})
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		socket.HandleWebSocket(w, r)
	})
	http.HandleFunc("/webhook", webhook.HandleWebhook)
	// http.HandleFunc("/send", socket.HandleSend)
	http.ListenAndServe(":8000", nil)
}
