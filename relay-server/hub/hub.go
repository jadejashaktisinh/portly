package hub

import (
	"errors"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var ErrClientNotFound = errors.New("client not found")

type Client struct {
	LocalUrl string
	Conn     *websocket.Conn
}

type LocalResponse struct {
	ResponseId string
	StatusCode int
	Header     http.Header
	Body       string
}

type Message struct {
	Id         string
	Body       string
	LocalUrl   string
	Header     http.Header
	StatusCode int
	RequestId  string
}

var (
	clients       = map[string]Client{}
	clientsMu     sync.RWMutex
	clientWriteMu sync.Mutex

	localResponses   = map[string]chan LocalResponse{}
	localResponsesMu sync.Mutex
)

func AddClient(id string, localUrl string, conn *websocket.Conn) bool {
	clientsMu.Lock()
	defer clientsMu.Unlock()
	if _, exists := clients[id]; exists {
		return false
	}
	clients[id] = Client{LocalUrl: localUrl, Conn: conn}
	return true
}

func RemoveClient(id string, conn *websocket.Conn) {
	clientsMu.Lock()
	defer clientsMu.Unlock()
	if current, ok := clients[id]; ok && current.Conn == conn {
		delete(clients, id)
		log.Println("client disconnected")
	}
}

func GetClient(id string) (Client, bool) {
	clientsMu.RLock()
	defer clientsMu.RUnlock()
	client, ok := clients[id]
	return client, ok
}

func SendToClient(id string, msg Message) error {
	client, ok := GetClient(id)
	if !ok {
		return ErrClientNotFound
	}
	clientWriteMu.Lock()
	defer clientWriteMu.Unlock()
	return client.Conn.WriteJSON(msg)
}

func CreateLocalResponseChannel(requestId string) <-chan LocalResponse {
	localResponsesMu.Lock()
	defer localResponsesMu.Unlock()
	ch := make(chan LocalResponse, 1)
	localResponses[requestId] = ch
	return ch
}

func GetLocalResponseChannel(requestId string) <-chan LocalResponse {
	localResponsesMu.Lock()
	defer localResponsesMu.Unlock()
	return localResponses[requestId]
}

func ResolveLocalResponse(response LocalResponse) {
	localResponsesMu.Lock()
	defer localResponsesMu.Unlock()
	if ch, ok := localResponses[response.ResponseId]; ok {
		select {
		case ch <- response:
		default:
		}
	}
}

func RemoveLocalResponse(requestId string) {
	localResponsesMu.Lock()
	defer localResponsesMu.Unlock()
	delete(localResponses, requestId)
}
