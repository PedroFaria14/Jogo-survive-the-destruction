package controller // Pacote controller para acessar tipos definidos em hub.go

import (
	"encoding/json"
	"game-backend/models" // Necessário para models.Command
	"log"
	"runtime/debug"
	"time"

	"github.com/gorilla/websocket" // Necessário para ReadMessage/WriteMessage
)

// Close encerra o transporte uma única vez, sem bloquear o loop do Hub.
func (c *Client) Close() {
	c.closeOnce.Do(func() {
		c.closed.Store(true)
		c.Hub.ConnCount.Add(-1)
		_ = c.Conn.Close()
	})
}

func (c *Client) disconnect() {
	c.Close()
	c.unregisterOnce.Do(func() {
		select {
		case c.Hub.Unregister <- c:
		case <-c.Hub.stopping:
		}
	})
}

// Timeouts de conexão WebSocket: pings mantêm a conexão viva e detectam
// conexões mortas (rede partida, máquina suspensa).
const (
	writeWait  = 10 * time.Second // Limite por escrita (mensagem ou ping)
	pongWait   = 60 * time.Second // Limite sem pong do cliente
	pingPeriod = (pongWait * 9) / 10
)

// ReadPump lida com as mensagens de entrada do cliente (EXPORTADA)
func ReadPump(c *Client) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("PANIC capturado em ReadPump (%s): %v\n%s", c.PlayerID, r, debug.Stack())
		}
		// Desregistra o cliente e fecha a conexão quando a função termina.
		// closeOnce garante que isso aconteça exatamente uma vez (os dois
		// pumps podem sair em qualquer ordem).
		c.disconnect()
	}()

	// Limite de tamanho de mensagem (evita mensagens gigantes) e keepalive:
	// o deadline de leitura é renovado a cada pong do cliente.
	c.Conn.SetReadLimit(4096)
	_ = c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		return c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		// Lê a mensagem do cliente. O comando é o único tipo de mensagem esperado.
		kind, message, err := c.Conn.ReadMessage()
		if err != nil {
			// Geralmente ocorre quando o cliente fecha a conexão ou há um erro de leitura.
			if !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) && !c.closed.Load() {
				log.Printf("websocket read failed: %v", err)
			}
			break
		}
		if kind != websocket.TextMessage {
			continue
		}

		var cmd models.Command
		err = json.Unmarshal(message, &cmd)
		if err != nil {
			log.Println("Invalid command format:", err)
			continue
		}
		if cmd.Type != "join" && cmd.Type != "input" && cmd.Type != "restart" {
			continue
		}

		// Cria o PlayerCommand, anexando o client (e portanto sua sala) atual.
		playerCmd := PlayerCommand{
			Client: c,
			Cmd:    cmd,
		}
		// Envia o comando para o Hub processar
		select {
		case c.Hub.Command <- playerCmd:
		case <-c.Hub.stopping:
			return
		}
	}
}

// WritePump lida com o envio de mensagens (GameState) para o cliente (EXPORTADA)
func WritePump(c *Client) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		if r := recover(); r != nil {
			log.Printf("PANIC capturado em WritePump (%s): %v\n%s", c.PlayerID, r, debug.Stack())
		}
		ticker.Stop()
		// Encerra apenas uma vez, independentemente de qual pump sair primeiro.
		c.disconnect()
	}()

	for {
		select {
		case <-c.Hub.stopping:
			_ = c.Conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseGoingAway, "server shutdown"), time.Now().Add(writeWait))
			return
		case message, ok := <-c.Send:
			_ = c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// O Hub fechou o canal. Envia close e encerra.
				_ = c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			// Envia o GameState serializado (que veio do canal Send do Hub)
			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Println("WritePump error:", err)
				return
			}
		case <-ticker.C:
			// Ping periódico: mantém a conexão viva e detecta clientes mortos.
			_ = c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
