package main

import (
	"fmt"
	"jadejashaktisinh/relay-server/socket"
	"net/http"
)

func main() {

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, World!")
	})
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		socket.HandleWebSocket(w, r)
	})
	http.HandleFunc("/send", socket.HandleSend)
	http.ListenAndServe(":8000", nil)
}
