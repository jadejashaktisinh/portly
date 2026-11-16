package socket

import (
	"fmt"
	"jadejashaktisinh/relay-server/hub"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	localUrl := r.URL.Query().Get("localUrl")
	if id == "" || localUrl == "" {
		http.Error(w, "id and localUrl params required", http.StatusBadRequest)
		return
	}
	fmt.Printf("WebSocket connection request for id: %s and localUrl: %s\n", id, localUrl)
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrade error:", err)
		return
	}
	defer func() {
		conn.Close()
		hub.RemoveClient(id)
	}()

	hub.AddClient(id, localUrl, conn)
	log.Println("connected:", id)

	for {

		msg := hub.Message{}
		err := conn.ReadJSON(&msg)
		if err != nil {
			log.Println("read error:", err)
			return
		}
		// jsonData := message{Id: id, Body: msg.Body, LocalUrl: localUrl}
		log.Printf("[%s] json : %s", id, msg.RequestId)
		ch := hub.GetLocalResponseChannel(msg.RequestId)
		if ch == nil {
			log.Printf("no pending request for requestId: %s", msg.RequestId)
			continue
		}
		log.Printf("got response from local server for requestId: %s and id: %s", msg.RequestId, id)
		hub.UpdateLocalResponseChannel(hub.LocalResponse{ResponseId: msg.RequestId, StatusCode: 200, Body: msg.Body, Header: msg.Header})
	}

}
