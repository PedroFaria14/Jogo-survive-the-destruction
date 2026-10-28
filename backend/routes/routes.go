package routes

import (
	"encoding/json"
	"game-backend/controller"
	"game-backend/models"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Config agrupa opções de configuração das rotas.
type Config struct {
	AllowedOrigins string // Origens permitidas no WebSocket, separadas por vírgula
}

// handshakeLimiter limita tentativas de handshake do WebSocket por IP
// (janela deslizante), protegendo contra abuso de conexões.
type handshakeLimiter struct {
	mu          sync.Mutex
	perIP       map[string][]time.Time
	max         int
	win         time.Duration
	nextCleanup time.Time
}

func newHandshakeLimiter(max int, win time.Duration) *handshakeLimiter {
	return &handshakeLimiter{
		perIP: make(map[string][]time.Time),
		max:   max,
		win:   win,
	}
}

// allow registra a tentativa e retorna se ela pode prosseguir dentro do limite.
func (l *handshakeLimiter) allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	cutoff := now.Add(-l.win)
	if !now.Before(l.nextCleanup) {
		for key, attempts := range l.perIP {
			if len(attempts) == 0 || !attempts[len(attempts)-1].After(cutoff) {
				delete(l.perIP, key)
			}
		}
		l.nextCleanup = now.Add(l.win)
	}
	stamps := l.perIP[ip]
	filtered := stamps[:0]
	for _, t := range stamps {
		if t.After(cutoff) {
			filtered = append(filtered, t)
		}
	}
	if len(filtered) >= l.max {
		l.perIP[ip] = filtered
		return false
	}
	l.perIP[ip] = append(filtered, now)
	return true
}

// clientIP extrai o IP do cliente. Prefere o primeiro hop do X-Forwarded-For
// (quando atrás do proxy do Render); senão usa o RemoteAddr.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if ip := strings.TrimSpace(parts[0]); ip != "" {
			return ip
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

type websocketHandler struct {
	hub      *controller.Hub
	upgrader websocket.Upgrader
	limiter  *handshakeLimiter
}

// newUpgrader cria um Upgrader com validação de origem.
func newUpgrader(allowed string) websocket.Upgrader {
	allowedSet := parseOrigins(allowed)

	// Se a lista estiver vazia, não aceita nenhuma origem (mais seguro).
	return websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			// Requisições sem header Origin (ex.: HTTP 1.0 / ferramentas) são negadas
			// a menos que estejam explicitamente permitidas.
			return origin != "" && allowedSet[origin]
		},
	}
}

// parseOrigins converte a lista separada por vírgulas em um conjunto.
func parseOrigins(allowed string) map[string]bool {
	allowedSet := make(map[string]bool)
	for _, origin := range strings.Split(allowed, ",") {
		origin = strings.TrimRight(strings.TrimSpace(origin), "/")
		if origin != "" {
			allowedSet[origin] = true
		}
	}
	return allowedSet
}

// withCORS aplica os headers CORS às respostas das rotas de API, ecoando a
// origem da requisição apenas quando ela estiver na lista permitida (mesma
// lista usada na validação do WebSocket, evitando duas listas divergentes).
func withCORS(allowedSet map[string]bool, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		setSecurityHeaders(w)
		origin := r.Header.Get("Origin")
		if allowedSet[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Add("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next(w, r)
	}
}

// setSecurityHeaders aplica headers de hardening às respostas HTTP/WS.
func setSecurityHeaders(w http.ResponseWriter) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("Cache-Control", "no-store")
}

// MaxConnections permite que o limite global de conexões do WebSocket seja
// injetado nos testes (serveWs usa hub.ConnCount diretamente).
func (handler *websocketHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	hub := handler.hub
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	// Rate limit por IP antes do upgrade (mitiga DoS por handshakes).
	if !handler.limiter.allow(clientIP(r)) {
		http.Error(w, "Muitas tentativas de conexão. Tente novamente em instantes.", http.StatusTooManyRequests)
		return
	}
	// Limite global de conexões simultâneas.
	if !hub.ReserveConnection() {
		http.Error(w, "Servidor cheio. Tente novamente em instantes.", http.StatusServiceUnavailable)
		return
	}

	setSecurityHeaders(w)
	conn, err := handler.upgrader.Upgrade(w, r, nil)
	if err != nil {
		hub.ConnCount.Add(-1)
		log.Println("Erro ao fazer upgrade para WebSocket:", err)
		return
	}

	hub.ServeClient(conn)
}

func getScoresHandler(hub *controller.Hub, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}

	scores, err := hub.ScoreService.GetTopScores()
	if err != nil {
		log.Printf("Erro ao buscar placares: %v", err)
		http.Error(w, "Erro interno ao buscar placares", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(scores)
}

func getHealthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func getConfigHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models.GetGameConfig())
}

// NewHandler configura rotas isoladas, sem alterar o mux global de net/http.
func NewHandler(hub *controller.Hub, cfg Config) http.Handler {
	mux := http.NewServeMux()
	allowedSet := parseOrigins(cfg.AllowedOrigins)
	ws := &websocketHandler{hub: hub, upgrader: newUpgrader(cfg.AllowedOrigins), limiter: newHandshakeLimiter(20, time.Minute)}

	// 1. Rota WebSocket
	mux.Handle("/ws", ws)

	// 2. Rota de API para o Placar
	mux.HandleFunc("/api/scores", withCORS(allowedSet, func(w http.ResponseWriter, r *http.Request) {
		getScoresHandler(hub, w, r)
	}))

	// 3. Rota de API para a configuração compartilhada do jogo
	mux.HandleFunc("/api/config", withCORS(allowedSet, getConfigHandler))

	// 4. Rota de health check (prontidão do servidor para o front)
	mux.HandleFunc("/api/health", withCORS(allowedSet, getHealthHandler))

	log.Printf("Rotas /ws, /api/scores, /api/config e /api/health configuradas com origens permitidas: %q.", cfg.AllowedOrigins)
	return mux
}
