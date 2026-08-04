package main

import (
	"bufio"
	"game-backend/controller"
	"game-backend/routes"
	"game-backend/service"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// findEnvFile localiza o arquivo .env independentemente do diretório de
// trabalho. Tenta, nesta ordem: o CWD atual, o diretório pai e o diretório
// ancestral do arquivo de origem (raiz do repositório). Retorna o primeiro
// que existir, ou uma string vazia se nenhum for encontrado.
func findEnvFile() string {
	// Ancorado na localização deste arquivo de código, não no CWD.
	_, thisFile, _, _ := runtime.Caller(0)
	root := filepath.Dir(filepath.Dir(thisFile)) // sobe de backend/ para a raiz do repo

	candidates := []string{
		".env",                      // rodando a partir da raiz
		"../.env",                   // rodando a partir de backend/
		filepath.Join(root, ".env"), // raiz do repositório
	}

	for _, path := range candidates {
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return path
		}
	}
	return ""
}

// loadEnv lê um arquivo .env simples (formato CHAVE=VALOR) localizado na raiz
// do projeto. Variáveis já definidas no ambiente do sistema têm prioridade.
func loadEnv(path string) error {
	if path == "" {
		return os.ErrNotExist
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if _, ok := os.LookupEnv(key); !ok {
			_ = os.Setenv(key, value)
		}
	}
	return scanner.Err()
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	// 1. CONFIGURAÇÃO DO POSTGRESQL A PARTIR DE VARIÁVEIS DE AMBIENTE
	if err := loadEnv(findEnvFile()); err != nil {
		// O .env é opcional; as vars podem vir do ambiente do sistema.
		log.Printf("Aviso: arquivo .env não encontrado (%v). Usando ambiente do sistema.", err)
	}
	connStr := getenv("DATABASE_URL", "")

	if connStr == "" {
		log.Fatal("A variável de ambiente DATABASE_URL é obrigatória. Defina-a ou crie um arquivo .env a partir de .env.example.")
	}

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
	routes.InitRoutes(hub, routes.Config{
		AllowedOrigins: getenv("ALLOWED_ORIGINS", "http://localhost:5173"),
	})

	// 4. Inicia o servidor HTTP
	addr := getenv("HTTP_ADDR", ":8080")
	log.Printf("Servidor iniciado em %s", addr)
	err = http.ListenAndServe(addr, nil)
	if err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}