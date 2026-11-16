package webhook

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"jadejashaktisinh/relay-server/hub"
	"log"
	"net/http"
	"strings"
)

func HandleWebhook(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello, World! from webhook domain "+r.Host)
	hostname := r.Host
	if hostname == "" {
		http.Error(w, "Host header is required", http.StatusBadRequest)
		return
	}
	slug, domain, found := strings.Cut(hostname, ".")

	fmt.Printf("Received webhook request for slug: %s, domain: %s found: %t\n", slug, domain, found)
	if !found {
		http.Error(w, "Invalid Host header format", http.StatusBadRequest)
		return
	}
	HandleSend(w, r)
}

func HandleSend(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	log.Println("got new message:")

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

	clientCh := hub.GetClientChannel(id)

	reqId, _ := GenerateRandomID(8)
	if clientCh == nil {
		http.Error(w, "client not found", http.StatusNotFound)
		return
	}

	select {
	case client := <-clientCh:
		log.Printf("generated requestId: %s for id: %s", reqId, id)

		// create channel BEFORE sending so the response handler never misses it
		hub.CreateLocalResponseChannel(reqId)

		jsonData := hub.Message{Id: id, Body: string(body), LocalUrl: client.LocalUrl, Header: header, RequestId: reqId}
		log.Println("sending message to:", id, "json:", jsonData)
		err := client.Conn.WriteJSON(jsonData)
		clientCh <- client
		if err != nil {
			log.Println("write error:", err)
			hub.LocalResponsesMu.Lock()
			delete(hub.LocalResponses, reqId)
			hub.LocalResponsesMu.Unlock()
			http.Error(w, "failed to send message", http.StatusInternalServerError)
			return
		}

		// block until CLI sends the response back
		localResponse := <-hub.GetLocalResponseChannel(reqId)
		hub.LocalResponsesMu.Lock()
		delete(hub.LocalResponses, reqId)
		hub.LocalResponsesMu.Unlock()

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
