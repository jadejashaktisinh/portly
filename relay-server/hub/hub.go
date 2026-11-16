package hub

import (
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

type client = struct {
	LocalUrl string
	Conn     *websocket.Conn
}

type LocalResponse = struct {
	ResponseId string
	StatusCode int
	Header     http.Header
	Body       string
	Writer     http.ResponseWriter
}

type Message struct {
	Id        string
	Body      string
	LocalUrl  string
	Header    http.Header
	RequestId string
}

var (
	clients          = map[string]chan client{}
	LocalResponses   = map[string]chan LocalResponse{}
	clientsMu        sync.Mutex
	LocalResponsesMu sync.Mutex
)

func AddClient(id string, localUrl string, conn *websocket.Conn) {
	clientsMu.Lock()
	defer clientsMu.Unlock()
	clients[id] = make(chan client, 1)
	clients[id] <- client{LocalUrl: localUrl, Conn: conn}
}

func RemoveClient(id string) {
	clientsMu.Lock()
	clientCh, ok := clients[id]
	if ok {
		close(clientCh)
	}
	delete(clients, id)
	clientsMu.Unlock()
	log.Println("disconnected:", id)
}

func GetClientChannel(id string) chan client {
	clientsMu.Lock()
	defer clientsMu.Unlock()
	return clients[id]
}

func CreateLocalResponseChannel(requestId string) {
	LocalResponsesMu.Lock()
	defer LocalResponsesMu.Unlock()
	LocalResponses[requestId] = make(chan LocalResponse, 1)
}

func GetLocalResponseChannel(requestId string) chan LocalResponse {
	LocalResponsesMu.Lock()
	defer LocalResponsesMu.Unlock()
	return LocalResponses[requestId]
}

func UpdateLocalResponseChannel(response LocalResponse) {
	LocalResponsesMu.Lock()
	ch, ok := LocalResponses[response.ResponseId]
	if ok {
		ch <- response
	}
	LocalResponsesMu.Unlock()
}
