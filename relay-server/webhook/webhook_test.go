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

func mockValidator(id string) func(context.Context, string) (db.User, bool, error) {
	if id == "bad" {
		return func(context.Context, string) (db.User, bool, error) { return db.User{}, false, nil }
	}
	return func(context.Context, string) (db.User, bool, error) {
		return db.User{ID: "user-1"}, true, nil
	}
}

func withMocks(t *testing.T, key string) *db.WebhookEvent {
	t.Helper()
	validateAPIKey = mockValidator(key)
	var recorded db.WebhookEvent
	recordEvent = func(_ context.Context, event db.WebhookEvent) error {
		recorded = event
		return nil
	}
	t.Cleanup(func() {
		validateAPIKey = db.FindUserByAPIKey
		recordEvent = db.RecordWebhookEvent
	})
	return &recorded
}

func TestHandleWebhookPropagatesLocalResponse(t *testing.T) {
	logged := withMocks(t, "test-key")

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

	if logged.RequestID == "" {
		t.Error("event request id was not recorded")
	}
	if logged.UserID != "user-1" {
		t.Errorf("event user id = %q, want user-1", logged.UserID)
	}
	if logged.StatusCode != http.StatusCreated {
		t.Errorf("event status = %d, want %d", logged.StatusCode, http.StatusCreated)
	}
	if logged.Body != `{"event":"test"}` {
		t.Errorf("event body = %q", logged.Body)
	}
	if logged.Host != "demo.localhost" || logged.Path != "/webhook" || logged.Slug != "demo" {
		t.Errorf("event request info = %+v", logged)
	}
	if logged.Error != "" {
		t.Errorf("event error = %q, want empty", logged.Error)
	}
}

func TestHandleWebhookRejectsInvalidAPIKey(t *testing.T) {
	validateAPIKey = mockValidator("bad")
	recordCalled := false
	recordEvent = func(context.Context, db.WebhookEvent) error {
		recordCalled = true
		return nil
	}
	t.Cleanup(func() {
		validateAPIKey = db.FindUserByAPIKey
		recordEvent = db.RecordWebhookEvent
	})

	req := httptest.NewRequest(http.MethodPost, "/webhook?id=bad", nil)
	req.Host = "demo.localhost"
	rec := httptest.NewRecorder()
	HandleWebhook(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if recordCalled {
		t.Error("invalid API key request should not be logged")
	}
}

func TestHandleWebhookRejectsOversizedBody(t *testing.T) {
	logged := withMocks(t, "test-key")

	body := strings.NewReader(strings.Repeat("x", (1<<20)+1))
	req := httptest.NewRequest(http.MethodPost, "/webhook?id=test-key", body)
	req.Host = "demo.localhost"
	rec := httptest.NewRecorder()
	HandleWebhook(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusRequestEntityTooLarge)
	}
	if logged.StatusCode != http.StatusRequestEntityTooLarge || logged.Error != "request body too large" {
		t.Errorf("event = %+v", logged)
	}
}
