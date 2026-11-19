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

var (
	validateAPIKey = db.FindUserByAPIKey
	recordEvent    = db.RecordWebhookEvent
)

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusRecorder) Write(body []byte) (int, error) {
	if s.status == 0 {
		s.WriteHeader(http.StatusOK)
	}
	return s.ResponseWriter.Write(body)
}

func HandleWebhook(w http.ResponseWriter, r *http.Request) {
	recorder := &statusRecorder{ResponseWriter: w}
	started := time.Now()
	var slug, userID, failure string
	var body []byte
	var reqID string

	defer func() {
		if userID == "" {
			return
		}
		event := db.WebhookEvent{
			RequestID:  reqID,
			UserID:     userID,
			Method:     r.Method,
			Host:       r.Host,
			Slug:       slug,
			Path:       r.URL.Path,
			Headers:    r.Header,
			Body:       string(body),
			StatusCode: recorder.status,
			DurationMs: int(time.Since(started).Milliseconds()),
			Error:      failure,
		}
		if err := recordEvent(context.Background(), event); err != nil {
			log.Printf("failed to record webhook event: %v", err)
		}
	}()

	fail := func(status int, message string) {
		failure = message
		http.Error(recorder, message, status)
	}

	hostname := r.Host
	if hostname == "" {
		fail(http.StatusBadRequest, "Host header is required")
		return
	}
	var domain string
	var found bool
	slug, domain, found = strings.Cut(hostname, ".")
	log.Printf("Received webhook request for slug: %s, domain: %s found: %t", slug, domain, found)
	if !found {
		fail(http.StatusBadRequest, "Invalid Host header format")
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		fail(http.StatusBadRequest, "id query parameter is required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), validationTime)
	defer cancel()
	user, valid, err := validateAPIKey(ctx, id)
	if err != nil {
		log.Printf("API key validation failed: %v", err)
		fail(http.StatusServiceUnavailable, "could not validate API key")
		return
	}
	if !valid {
		fail(http.StatusUnauthorized, "invalid API key")
		return
	}
	userID = user.ID

	r.Body = http.MaxBytesReader(recorder, r.Body, maxBodySize)
	body, err = io.ReadAll(r.Body)
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			fail(http.StatusRequestEntityTooLarge, "request body too large")
			return
		}
		fail(http.StatusBadRequest, "failed to read request body")
		return
	}
	defer r.Body.Close()
	log.Printf("received webhook request (%d bytes)", len(body))

	reqID, err = GenerateRandomID(8)
	if err != nil {
		log.Printf("failed to generate request id: %v", err)
		fail(http.StatusInternalServerError, "internal server error")
		return
	}

	client, ok := hub.GetClient(id)
	if !ok {
		fail(http.StatusNotFound, "client not connected")
		return
	}

	responseCh := hub.CreateLocalResponseChannel(reqID)
	defer hub.RemoveLocalResponse(reqID)

	jsonData := hub.Message{
		Id:        id,
		Body:      string(body),
		LocalUrl:  client.LocalUrl,
		Header:    r.Header,
		RequestId: reqID,
	}
	log.Printf("forwarding request %s to %s", reqID, client.LocalUrl)
	if err := hub.SendToClient(id, jsonData); err != nil {
		log.Printf("failed to forward request %s: %v", reqID, err)
		fail(http.StatusBadGateway, "failed to send message")
		return
	}

	select {
	case localResponse := <-responseCh:
		for key, values := range localResponse.Header {
			for _, value := range values {
				recorder.Header().Add(key, value)
			}
		}
		recorder.WriteHeader(localResponse.StatusCode)
		recorder.Write([]byte(localResponse.Body))
	case <-time.After(localTimeout):
		log.Printf("request %s timed out waiting for local server", reqID)
		fail(http.StatusGatewayTimeout, "local server timed out")
	}
}

func GenerateRandomID(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
