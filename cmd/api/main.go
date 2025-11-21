package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"loginbackend/config"
	"loginbackend/features/auth"
	"loginbackend/features/users"
	"loginbackend/internal/database"
	httpPlatform "loginbackend/internal/http"
	"loginbackend/pkg/utils"

	// Adicione o import do Redis aqui (necessário para o tipo e inicialização interna)

	_ "loginbackend/docs"

	httpSwagger "github.com/swaggo/http-swagger"
)

// @title Login Backend API
// @version 1.0
// @description API de autenticação e gestão de usuários com PostgreSQL e Snowflake ID
// @host localhost:8080
// @BasePath /
func main() {
	cfg := config.Load()

	// Inicializar Snowflake ID
	if err := utils.InitSnowflake(1); err != nil {
		log.Fatal("Erro ao inicializar Snowflake:", err)
	}

	// 1. Conexão com PostgreSQL
	db, err := database.NewPostgres(cfg.GetConnectionString())
	if err != nil {
		log.Fatal(err)
	}

	redisClient, err := database.NewRedis(cfg.RedisHost, cfg.RedisPort, cfg.RedisPassword)
	if err != nil {
		log.Fatal("Erro ao conectar no Redis: ", err)
	}

	// Rodar migrations
	if err := database.RunMigrations(db, "./migrations"); err != nil {
		log.Fatal(err)
	}

	r := httpPlatform.NewRouter(cfg, redisClient)

	// Swagger
	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("http://localhost:8080/swagger/doc.json"),
	))

	// Registrar feature de autenticação
	authRepo := auth.NewRepository(db)
	authService := auth.NewService(authRepo, auth.Config{
		JWTSecret:     cfg.JWTSecret,
		AccessExpiry:  5 * time.Minute,
		RefreshExpiry: 15 * time.Minute,
	})
	authHandler := auth.NewHandler(authService)

	authPath, authRoutes := auth.Routes(authHandler, redisClient)
	r.Route(authPath, authRoutes)

	// Registrar feature de usuários
	usersRepo := users.NewRepository(db)
	usersService := users.NewService(usersRepo)
	usersHandler := users.NewHandler(usersService)

	usersPath, usersRoutes := users.Routes(usersHandler, cfg.JWTSecret)
	r.Route(usersPath, usersRoutes)

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		// Verifica DB
		if err := db.Ping(); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{"status": "db_unhealthy"})
			return
		}
		// Verifica Redis
		if err := redisClient.Ping(r.Context()).Err(); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{"status": "redis_unhealthy"})
			return
		}

		json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
	})

	log.Println("🚀 API rodando em http://localhost:8080")
	log.Println("📚 Swagger disponível em http://localhost:8080/swagger/index.html")
	log.Printf("📁 Database: PostgreSQL + Redis")

	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatal(err)
	}
}
