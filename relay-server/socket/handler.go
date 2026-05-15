package socket

import (
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

var (
	clients   = map[string]*websocket.Conn{}
	clientsMu sync.Mutex
)

func HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "id param required", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrade error:", err)
		return
	}
	defer func() {
		clientsMu.Lock()
		delete(clients, id)
		clientsMu.Unlock()
		conn.Close()
		log.Println("disconnected:", id)
	}()

	clientsMu.Lock()
	clients[id] = conn
	clientsMu.Unlock()
	log.Println("connected:", id)

	for {
		msgType, msg, err := conn.ReadMessage()
		if err != nil {
			break
		}
		log.Printf("[%s] received: %s", id, msg)
		conn.WriteMessage(msgType, msg)
	}
}

// POST /send?id=abc&msg=hello
func HandleSend(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("id")
	msg := r.FormValue("msg")
	if id == "" || msg == "" {
		http.Error(w, "id and msg params required", http.StatusBadRequest)
		return
	}

	clientsMu.Lock()
	conn, ok := clients[id]
	clientsMu.Unlock()

	if !ok {
		http.Error(w, "client not found", http.StatusNotFound)
		return
	}

	conn.WriteMessage(websocket.TextMessage, []byte(msg))
	w.Write([]byte("sent"))
}
