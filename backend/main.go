package main

import (
	"game-backend/controller"
	"game-backend/routes"
	"game-backend/service"
	"log"
	"net/http"
)

func main() {
	// 1. CONFIGURAÇÃO DO POSTGRESQL
	// A string de conexão foi atualizada com a senha e o nome do banco de dados (survive_the_destruction)
	const connStr = "user=postgres password=1234567 dbname=survive_the_destruction host=localhost sslmode=disable"

	scoreService, err := service.NewPostgresScoreService(connStr)
	if err != nil {
		// Loga o erro exato. Se houver falha de conexão, pare o aplicativo.
		log.Fatalf("Falha ao iniciar o ScoreService: %v", err)
	}

	// Fechará a conexão com o DB quando a função main() terminar.
	defer scoreService.(*service.PostgresScoreService).DB.Close()

	// 2. Cria e Inicia o Hub (o loop principal do jogo)
	// Passa o serviço de placar para o Hub
	hub := controller.NewHub(scoreService)
	go hub.Run()

	// 3. Configura todas as rotas do aplicativo, delegando ao pacote routes
	routes.InitRoutes(hub)

	// 4. Inicia o servidor HTTP
	log.Println("Servidor iniciado em :8080")
	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}
