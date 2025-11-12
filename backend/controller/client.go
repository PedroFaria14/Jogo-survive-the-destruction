package controller // Pacote controller para acessar tipos definidos em hub.go

import (
	"encoding/json"
	"game-backend/models" // Necessário para models.Command
	"log"

	"github.com/gorilla/websocket" // Necessário para ReadMessage/WriteMessage
)

// ReadPump lida com as mensagens de entrada do cliente (EXPORTADA)
func ReadPump(c *Client) {
	defer func() {
		// Desregistra o cliente e fecha a conexão quando a função termina
		c.Hub.Unregister <- c
		c.Conn.Close()
	}()

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

		// Cria o PlayerCommand, anexando o ID do jogador atual (a correção mais importante)
		playerCmd := PlayerCommand{
			PlayerID: c.PlayerID,
			Cmd:      cmd,
		}
		// Envia o comando para o Hub processar
		c.Hub.Command <- playerCmd
	}
}

// WritePump lida com o envio de mensagens (GameState) para o cliente (EXPORTADA)
func WritePump(c *Client) {
	defer func() {
		c.Conn.Close()
	}()
	for message := range c.Send {
		// Envia o GameState serializado (que veio do canal Broadcast do Hub)
		err := c.Conn.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			log.Println("WritePump error:", err)
			return
		}
	}
	// O Hub fechou o canal. Encerra a conexão.
	c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
}
