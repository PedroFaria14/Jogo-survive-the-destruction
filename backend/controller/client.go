package controller // Pacote controller para acessar tipos definidos em hub.go

import (
	"encoding/json"
	"game-backend/models" // Necessário para models.Command
	"log"
	"runtime/debug"
	"time"

	"github.com/gorilla/websocket" // Necessário para ReadMessage/WriteMessage
)

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
		c.closed.Store(true)
		// Desregistra o cliente e fecha a conexão quando a função termina.
		// closeOnce garante que isso aconteça exatamente uma vez (os dois
		// pumps podem sair em qualquer ordem).
		c.closeOnce.Do(func() {
			c.Hub.ConnCount.Add(-1)
			c.Hub.Unregister <- c
			c.Conn.Close()
		})
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
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			// Geralmente ocorre quando o cliente fecha a conexão ou há um erro de leitura.
			log.Println("ReadPump error:", err)
			break
		}

		var cmd models.Command
		err = json.Unmarshal(message, &cmd)
		if err != nil {
			log.Println("Invalid command format:", err)
			continue
		}

		// Cria o PlayerCommand, anexando o client (e portanto sua sala) atual.
		playerCmd := PlayerCommand{
			Client: c,
			Cmd:    cmd,
		}
		// Envia o comando para o Hub processar
		c.Hub.Command <- playerCmd
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
		c.closed.Store(true)
		// Encerra apenas uma vez, independentemente de qual pump sair primeiro.
		c.closeOnce.Do(func() {
			c.Hub.ConnCount.Add(-1)
			c.Hub.Unregister <- c
			c.Conn.Close()
		})
	}()

	for {
		select {
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
