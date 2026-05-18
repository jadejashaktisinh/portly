package socket

import (
	"crypto/rand"
	"encoding/hex"
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

type LocalResponse = struct {
	responseId string
	StatusCode int
	Header     http.Header
	Body       string
	Writer     http.ResponseWriter
}

type message struct {
	Id        string
	Body      string
	LocalUrl  string
	Header    http.Header
	RequestId string
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

var (
	clients          = map[string]chan client{}
	LocalResponses   = map[string]chan LocalResponse{}
	clientsMu        sync.Mutex
	LocalResponsesMu sync.Mutex
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

		msg := message{}
		err := conn.ReadJSON(&msg)
		if err != nil {
			log.Println("read error:", err)
			return
		}
		// jsonData := message{Id: id, Body: msg.Body, LocalUrl: localUrl}
		log.Printf("[%s] json : %s", id, msg.RequestId)
		LocalResponsesMu.Lock()
		ch, ok := LocalResponses[msg.RequestId]
		LocalResponsesMu.Unlock()
		if !ok {
			log.Printf("no pending request for requestId: %s", msg.RequestId)
			continue
		}
		log.Printf("got response from local server for requestId: %s and id: %s", msg.RequestId, id)
		ch <- LocalResponse{responseId: msg.RequestId, StatusCode: 200, Body: msg.Body, Header: msg.Header}
	}

}

// POST /send?id=abc&msg=hello
func HandleSend(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	log.Println("got new message:")
	// msg := r.FormValue("msg")
	// if id == "" || msg == "" {
	// 	http.Error(w, "id and msg params required", http.StatusBadRequest)
	// 	return
	// }

	maxBodySize := int64(1024 * 1024)
	limitedReader := io.LimitReader(r.Body, maxBodySize)

	body, err := io.ReadAll(limitedReader)
	log.Println("got new message:", id, string(body))
	header := r.Header
	if err != nil {
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	clientsMu.Lock()
	clientCh, ok := clients[id]
	clientsMu.Unlock()
	reqId, _ := GenerateRandomID(8)
	if !ok {
		http.Error(w, "client not found", http.StatusNotFound)
		return
	}

	select {
	case client := <-clientCh:
		log.Printf("generated requestId: %s for id: %s", reqId, id)

		// create channel BEFORE sending so the response handler never misses it
		LocalResponsesMu.Lock()
		LocalResponses[reqId] = make(chan LocalResponse, 1)
		LocalResponsesMu.Unlock()

		jsonData := message{Id: id, Body: string(body), LocalUrl: client.LocalUrl, Header: header, RequestId: reqId}
		log.Println("sending message to:", id, "json:", jsonData)
		err := client.conn.WriteJSON(jsonData)
		clientCh <- client
		if err != nil {
			log.Println("write error:", err)
			LocalResponsesMu.Lock()
			delete(LocalResponses, reqId)
			LocalResponsesMu.Unlock()
			http.Error(w, "failed to send message", http.StatusInternalServerError)
			return
		}

		// block until CLI sends the response back
		localResponse := <-LocalResponses[reqId]
		LocalResponsesMu.Lock()
		delete(LocalResponses, reqId)
		LocalResponsesMu.Unlock()

		w.WriteHeader(localResponse.StatusCode)
		for key, values := range localResponse.Header {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}
		w.Write([]byte(localResponse.Body))
	default:
		http.Error(w, "client not connected", http.StatusBadRequest)
		return
	}
}

func GenerateRandomID(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
