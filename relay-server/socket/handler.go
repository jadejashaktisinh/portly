package socket

import (
	"io"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

type client = struct {
	LocalUrl string
	conn     *websocket.Conn
}

type message struct {
	Id       string
	Body     string
	LocalUrl string
	Header   http.Header
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

var (
	clients   = map[string]chan client{}
	clientsMu sync.Mutex
)

func HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	localUrl := r.URL.Query().Get("localUrl")
	if id == "" || localUrl == "" {
		http.Error(w, "id and localUrl params required", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrade error:", err)
		return
	}
	defer func() {
		clientsMu.Lock()
		clientCh, ok := clients[id]
		if ok {
			close(clientCh)
		}
		delete(clients, id)
		clientsMu.Unlock()
		conn.Close()
		log.Println("disconnected:", id)
	}()

	clientsMu.Lock()
	clients[id] = make(chan client, 1)
	clients[id] <- client{LocalUrl: localUrl, conn: conn}
	clientsMu.Unlock()
	log.Println("connected:", id)

	for {
		msgType, msg, err := conn.ReadMessage()
		if err != nil {
			break
		}
		log.Printf("[%s] received: %s", id, msg, "msgType:", msgType)
		// conn.WriteMessage(msgType, msg)
	}
}

// POST /send?id=abc&msg=hello
func HandleSend(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("id")
	// msg := r.FormValue("msg")
	// if id == "" || msg == "" {
	// 	http.Error(w, "id and msg params required", http.StatusBadRequest)
	// 	return
	// }
	body, err := io.ReadAll(r.Body)
	header := r.Header
	if err != nil {
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}

	clientsMu.Lock()
	clientCh, ok := clients[id]
	clientsMu.Unlock()

	if !ok {
		http.Error(w, "client not found", http.StatusNotFound)
		return
	}

	select {
	case client := <-clientCh:
		jsonData := message{Id: id, Body: string(body), LocalUrl: client.LocalUrl, Header: header}
		log.Println("sending message to:", id, "json:", jsonData)
		err := client.conn.WriteJSON(jsonData)
		clientCh <- client
		if err != nil {
			log.Println("write error:", err)
			http.Error(w, "failed to send message", http.StatusInternalServerError)
			return
		}
	default:
		http.Error(w, "client not connected", http.StatusBadRequest)
		return
	}
	w.Write([]byte("sent"))
}
