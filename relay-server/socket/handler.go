package socket

import (
	"context"
	"log"
	"net/http"
	"time"

	"jadejashaktisinh/relay-server/db"
	"jadejashaktisinh/relay-server/hub"

	"github.com/gorilla/websocket"
)

const clientIdleTimeout = 3 * time.Minute

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

var validateAPIKey = db.ValidateAPIKey

func HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	localUrl := r.URL.Query().Get("localUrl")
	if id == "" || localUrl == "" {
		http.Error(w, "id and localUrl params required", http.StatusBadRequest)
		return
	}
	log.Printf("WebSocket connection request for localUrl: %s", localUrl)

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	valid, err := validateAPIKey(ctx, id)
	if err != nil {
		log.Printf("API key validation failed: %v", err)
		http.Error(w, "could not validate API key", http.StatusServiceUnavailable)
		return
	}
	if !valid {
		http.Error(w, "invalid API key", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrade error:", err)
		return
	}
	defer func() {
		conn.Close()
		hub.RemoveClient(id, conn)
	}()

	conn.SetReadDeadline(time.Now().Add(clientIdleTimeout))
	conn.SetPingHandler(func(message string) error {
		conn.SetReadDeadline(time.Now().Add(clientIdleTimeout))
		return conn.WriteControl(websocket.PongMessage, []byte(message), time.Now().Add(time.Second))
	})
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(clientIdleTimeout))
		return nil
	})

	if !hub.AddClient(id, localUrl, conn) {
		conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "duplicate client"))
		return
	}
	log.Printf("client connected for %s", localUrl)

	for {
		msg := hub.Message{}
		if err := conn.ReadJSON(&msg); err != nil {
			log.Println("read error:", err)
			return
		}
		conn.SetReadDeadline(time.Now().Add(clientIdleTimeout))
		log.Printf("received response for requestId: %s", msg.RequestId)

		statusCode := msg.StatusCode
		if statusCode == 0 {
			statusCode = http.StatusOK
		}
		hub.ResolveLocalResponse(hub.LocalResponse{
			ResponseId: msg.RequestId,
			StatusCode: statusCode,
			Header:     msg.Header,
			Body:       msg.Body,
		})
	}
}
