package routes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestHandshakeLimiterVerificaJanela garante que o limite por IP bloqueia a
// tentativa excedente e não afeta outros IPs.
func TestHandshakeLimiter(t *testing.T) {
	l := newHandshakeLimiter(3, time.Minute)

	for i := 0; i < 3; i++ {
		if !l.allow("1.2.3.4") {
			t.Fatalf("tentativa %d deveria ser permitida", i+1)
		}
	}
	if l.allow("1.2.3.4") {
		t.Fatal("4ª tentativa do mesmo IP deveria ser bloqueada")
	}
	if !l.allow("5.6.7.8") {
		t.Fatal("IP diferente não deveria ser bloqueado")
	}
}

// TestHandshakeLimiterJanelaExpira garante que tentativas antigas saem da
// janela e o IP volta a ser permitido.
func TestHandshakeLimiterJanelaExpira(t *testing.T) {
	l := newHandshakeLimiter(1, time.Millisecond)
	if !l.allow("9.9.9.9") {
		t.Fatal("primeira tentativa deveria ser permitida")
	}
	if l.allow("9.9.9.9") {
		t.Fatal("segunda tentativa imediata deveria ser bloqueada")
	}
	time.Sleep(5 * time.Millisecond)
	if !l.allow("9.9.9.9") {
		t.Fatal("após expirar a janela, deveria ser permitido novamente")
	}
}

// TestHealthHandler garante que /api/health responde 200 com {"status":"ok"},
// sinalizando ao front que o servidor ligou (remove a tela de carregamento).
func TestHealthHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rec := httptest.NewRecorder()

	getHealthHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, esperado 200", rec.Code)
	}
	want := `{"status":"ok"}`
	if got := strings.TrimSpace(rec.Body.String()); got != want {
		t.Fatalf("body = %q, esperado %q", got, want)
	}
}
