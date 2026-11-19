package webhook

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"jadejashaktisinh/relay-server/db"
	"jadejashaktisinh/relay-server/hub"

	"github.com/gorilla/websocket"
)

func TestHandleWebhookPropagatesLocalResponse(t *testing.T) {
	validateAPIKey = func(context.Context, string) (bool, error) { return true, nil }
	defer func() { validateAPIKey = db.ValidateAPIKey }()

	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade: %v", err)
			return
		}
		defer conn.Close()
		if !hub.AddClient("test-key", "http://localhost:9999", conn) {
			t.Error("failed to register client")
			return
		}
		defer hub.RemoveClient("test-key", conn)

		var response hub.Message
		if err := conn.ReadJSON(&response); err != nil {
			t.Errorf("read CLI response: %v", err)
			return
		}
		hub.ResolveLocalResponse(hub.LocalResponse{
			ResponseId: response.RequestId,
			StatusCode: response.StatusCode,
			Header:     response.Header,
			Body:       response.Body,
		})
	}))
	defer server.Close()

	client, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer client.Close()

	deadline := time.Now().Add(2 * time.Second)
	for {
		if _, ok := hub.GetClient("test-key"); ok {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("client was not registered")
		}
		time.Sleep(10 * time.Millisecond)
	}

	go func() {
		var request hub.Message
		if err := client.ReadJSON(&request); err != nil {
			return
		}
		client.WriteJSON(hub.Message{
			RequestId:  request.RequestId,
			StatusCode: http.StatusCreated,
			Header:     http.Header{"X-Test": []string{"yes"}},
			Body:       "created",
		})
	}()

	req := httptest.NewRequest(http.MethodPost, "/webhook?id=test-key", strings.NewReader(`{"event":"test"}`))
	req.Host = "demo.localhost"
	rec := httptest.NewRecorder()
	HandleWebhook(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusCreated)
	}
	if got := rec.Header().Get("X-Test"); got != "yes" {
		t.Errorf("X-Test = %q, want yes", got)
	}
	if got := rec.Body.String(); got != "created" {
		t.Errorf("body = %q, want created", got)
	}
	if strings.Contains(rec.Body.String(), "Hello, World") {
		t.Error("greeting leaked into response")
	}
}

func TestHandleWebhookRejectsInvalidAPIKey(t *testing.T) {
	validateAPIKey = func(context.Context, string) (bool, error) { return false, nil }
	defer func() { validateAPIKey = db.ValidateAPIKey }()

	req := httptest.NewRequest(http.MethodPost, "/webhook?id=bad", nil)
	req.Host = "demo.localhost"
	rec := httptest.NewRecorder()
	HandleWebhook(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestHandleWebhookRejectsOversizedBody(t *testing.T) {
	validateAPIKey = func(context.Context, string) (bool, error) { return true, nil }
	defer func() { validateAPIKey = db.ValidateAPIKey }()

	body := strings.NewReader(strings.Repeat("x", (1<<20)+1))
	req := httptest.NewRequest(http.MethodPost, "/webhook?id=test-key", body)
	req.Host = "demo.localhost"
	rec := httptest.NewRecorder()
	HandleWebhook(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusRequestEntityTooLarge)
	}
}
