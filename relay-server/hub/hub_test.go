package hub

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestClientRegistry(t *testing.T) {
	first := &websocket.Conn{}
	second := &websocket.Conn{}

	if !AddClient("id", "http://first", first) {
		t.Fatal("first client was not registered")
	}
	if AddClient("id", "http://second", second) {
		t.Fatal("duplicate client was accepted")
	}

	client, ok := GetClient("id")
	if !ok || client.LocalUrl != "http://first" || client.Conn != first {
		t.Fatalf("GetClient returned %+v, %t", client, ok)
	}

	RemoveClient("id", second)
	if _, ok := GetClient("id"); !ok {
		t.Fatal("stale connection removed the active client")
	}

	RemoveClient("id", first)
	if _, ok := GetClient("id"); ok {
		t.Fatal("client was not removed")
	}
}

func TestLocalResponseChannel(t *testing.T) {
	ch := CreateLocalResponseChannel("request-1")
	ResolveLocalResponse(LocalResponse{ResponseId: "request-1", StatusCode: 202, Body: "ok"})

	select {
	case got := <-ch:
		if got.StatusCode != 202 || got.Body != "ok" {
			t.Fatalf("unexpected response: %+v", got)
		}
	case <-time.After(time.Second):
		t.Fatal("response was not delivered")
	}

	ResolveLocalResponse(LocalResponse{ResponseId: "request-1", Body: "duplicate"})
	RemoveLocalResponse("request-1")
	if GetLocalResponseChannel("request-1") != nil {
		t.Fatal("response channel was not removed")
	}
}

func TestConcurrentClientAccess(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			id := fmt.Sprintf("client-%d", i)
			conn := &websocket.Conn{}
			if !AddClient(id, "http://local", conn) {
				t.Errorf("could not register %s", id)
				return
			}
			if _, ok := GetClient(id); !ok {
				t.Errorf("could not read %s", id)
			}
			RemoveClient(id, conn)
		}(i)
	}
	wg.Wait()
}
