package webhook

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"jadejashaktisinh/relay-server/db"
	"jadejashaktisinh/relay-server/hub"
)

const (
	maxBodySize    = 1 << 20
	localTimeout   = 30 * time.Second
	validationTime = 5 * time.Second
)

var validateAPIKey = db.ValidateAPIKey

func HandleWebhook(w http.ResponseWriter, r *http.Request) {
	hostname := r.Host
	if hostname == "" {
		http.Error(w, "Host header is required", http.StatusBadRequest)
		return
	}
	slug, domain, found := strings.Cut(hostname, ".")
	log.Printf("Received webhook request for slug: %s, domain: %s found: %t", slug, domain, found)
	if !found {
		http.Error(w, "Invalid Host header format", http.StatusBadRequest)
		return
	}
	HandleSend(w, r)
}

func HandleSend(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "id query parameter is required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), validationTime)
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

	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
			return
		}
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()
	log.Printf("received webhook request (%d bytes)", len(body))

	reqId, err := GenerateRandomID(8)
	if err != nil {
		log.Printf("failed to generate request id: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	client, ok := hub.GetClient(id)
	if !ok {
		http.Error(w, "client not connected", http.StatusNotFound)
		return
	}

	responseCh := hub.CreateLocalResponseChannel(reqId)
	defer hub.RemoveLocalResponse(reqId)

	jsonData := hub.Message{
		Id:        id,
		Body:      string(body),
		LocalUrl:  client.LocalUrl,
		Header:    r.Header,
		RequestId: reqId,
	}
	log.Printf("forwarding request %s to %s", reqId, client.LocalUrl)
	if err := hub.SendToClient(id, jsonData); err != nil {
		log.Printf("failed to forward request %s: %v", reqId, err)
		http.Error(w, "failed to send message", http.StatusBadGateway)
		return
	}

	select {
	case localResponse := <-responseCh:
		for key, values := range localResponse.Header {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}
		w.WriteHeader(localResponse.StatusCode)
		w.Write([]byte(localResponse.Body))
	case <-time.After(localTimeout):
		log.Printf("request %s timed out waiting for local server", reqId)
		http.Error(w, "local server timed out", http.StatusGatewayTimeout)
	}
}

func GenerateRandomID(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
