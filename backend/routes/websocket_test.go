package routes

import (
	"context"
	"encoding/json"
	"game-backend/controller"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func startGameServer(t *testing.T) (*controller.Hub, *httptest.Server, context.CancelFunc) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	hub := controller.NewHub(nil)
	go hub.Run(ctx)
	server := httptest.NewServer(NewHandler(hub, Config{AllowedOrigins: "http://localhost:5173"}))
	t.Cleanup(func() {
		cancel()
		ctx, done := context.WithTimeout(context.Background(), 3*time.Second)
		defer done()
		if err := hub.Wait(ctx); err != nil {
			t.Error(err)
		}
		server.Close()
	})
	return hub, server, cancel
}

func dialGame(t *testing.T, server *httptest.Server) (*websocket.Conn, map[string]any) {
	t.Helper()
	conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http")+"/ws", http.Header{"Origin": {"http://localhost:5173"}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	var init map[string]any
	if err := conn.ReadJSON(&init); err != nil {
		t.Fatal(err)
	}
	if init["type"] != "init" || init["player_id"] == "" || init["room_id"] == "" {
		t.Fatalf("invalid init: %v", init)
	}
	return conn, init
}

func TestWebSocketJoinInputAndDisconnect(t *testing.T) {
	hub, server, _ := startGameServer(t)
	a, init := dialGame(t, server)
	b, other := dialGame(t, server)
	if init["room_id"] != other["room_id"] {
		t.Fatal("players must share room")
	}
	for _, msg := range []string{`broken`, `{"type":"unknown","right":true}`, `{"type":"join","name":" Pedro\n ","color":"#123456"}`, `{"type":"input","right":true}`} {
		if err := a.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
			t.Fatal(err)
		}
	}
	for {
		var state struct {
			Players     map[string]struct{ Name, Color string }
			RoomPlayers int `json:"room_players"`
		}
		if err := b.ReadJSON(&state); err != nil {
			t.Fatal(err)
		}
		p := state.Players[init["player_id"].(string)]
		if p.Name == "Pedro" && p.Color == "#123456" && state.RoomPlayers == 2 {
			break
		}
	}
	a.Close()
	for {
		var state struct {
			Players     map[string]json.RawMessage
			RoomPlayers int `json:"room_players"`
		}
		if err := b.ReadJSON(&state); err != nil {
			t.Fatal(err)
		}
		if state.RoomPlayers == 1 {
			if _, found := state.Players[init["player_id"].(string)]; found {
				t.Fatal("ghost player")
			}
			break
		}
	}
	if hub.ConnCount.Load() != 1 {
		t.Fatalf("connections = %d", hub.ConnCount.Load())
	}
}

func TestWebSocketRoomsAndShutdown(t *testing.T) {
	hub, server, cancel := startGameServer(t)
	var first string
	var conn *websocket.Conn
	for i := 0; i < controller.PlayersPerRoom+1; i++ {
		c, init := dialGame(t, server)
		conn = c
		room := init["room_id"].(string)
		if i == 0 {
			first = room
		}
		if i < controller.PlayersPerRoom && room != first {
			t.Fatal("room filled prematurely")
		}
		if i == controller.PlayersPerRoom && room == first {
			t.Fatal("room capacity exceeded")
		}
	}
	cancel()
	ctx, done := context.WithTimeout(context.Background(), 3*time.Second)
	defer done()
	if err := hub.Wait(ctx); err != nil {
		t.Fatal(err)
	}
	if hub.ConnCount.Load() != 0 {
		t.Fatal("connections leaked")
	}
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}
}

func TestWebSocketRejectsOriginAndReleasesReservation(t *testing.T) {
	hub, server, _ := startGameServer(t)
	_, response, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http")+"/ws", http.Header{"Origin": {"https://untrusted.example"}})
	if err == nil {
		t.Fatal("untrusted origin accepted")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusForbidden {
		t.Fatal(response.StatusCode)
	}
	if hub.ConnCount.Load() != 0 {
		t.Fatal("reservation leaked")
	}
}

func TestConnectionReservationIsBounded(t *testing.T) {
	hub := controller.NewHub(nil)
	var accepted atomic.Int64
	var tasks sync.WaitGroup
	for i := 0; i < controller.MaxClients*2; i++ {
		tasks.Add(1)
		go func() {
			defer tasks.Done()
			if hub.ReserveConnection() {
				accepted.Add(1)
			}
		}()
	}
	tasks.Wait()
	if accepted.Load() != controller.MaxClients {
		t.Fatalf("accepted %d reservations", accepted.Load())
	}
	if hub.ReserveConnection() {
		t.Fatal("capacity exceeded")
	}
}

func TestHandlersHaveIndependentOriginConfiguration(t *testing.T) {
	hub := controller.NewHub(nil)
	a := NewHandler(hub, Config{AllowedOrigins: "https://a.example/"})
	b := NewHandler(hub, Config{AllowedOrigins: "https://b.example"})
	for _, tc := range []struct {
		handler http.Handler
		origin  string
	}{{a, "https://a.example"}, {b, "https://b.example"}} {
		req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
		req.Header.Set("Origin", tc.origin)
		rec := httptest.NewRecorder()
		tc.handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK || rec.Header().Get("Access-Control-Allow-Origin") != tc.origin {
			t.Fatal("handler configuration leaked across instances")
		}
	}
}
