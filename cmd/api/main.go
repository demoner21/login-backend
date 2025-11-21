package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"loginbackend/config"
	"loginbackend/features/auth"
	"loginbackend/features/users"
	"loginbackend/internal/database"
	httpPlatform "loginbackend/internal/http"
	"loginbackend/pkg/utils" // Adicione esta linha

	// Import dos docs do Swagger
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

	db, err := database.NewPostgres(cfg.GetConnectionString())
	if err != nil {
		log.Fatal(err)
	}

	// Rodar migrations
	if err := database.RunMigrations(db, "./migrations"); err != nil {
		log.Fatal(err)
	}

	r := httpPlatform.NewRouter()

	// Configuração JWT
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "fallback-secret-change-in-production"
		log.Println("⚠️  JWT_SECRET não configurado, usando valor padrão")
	}

	// Swagger
	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("http://localhost:8080/swagger/doc.json"),
	))

	// Registrar feature de autenticação
	authRepo := auth.NewRepository(db)
	authService := auth.NewService(authRepo, auth.Config{
		JWTSecret:     jwtSecret,
		AccessExpiry:  5 * time.Minute,
		RefreshExpiry: 15 * time.Minute,
	})
	authHandler := auth.NewHandler(authService)
	authPath, authRoutes := auth.Routes(authHandler)
	r.Route(authPath, authRoutes)

	// Registrar feature de usuários
	usersRepo := users.NewRepository(db)
	usersService := users.NewService(usersRepo)
	usersHandler := users.NewHandler(usersService)
	usersPath, usersRoutes := users.Routes(usersHandler)
	r.Route(usersPath, usersRoutes)

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		if err := db.Ping(); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{"status": "unhealthy"})
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
	})

	log.Println("🚀 API rodando em http://localhost:8080")
	log.Println("📚 Swagger disponível em http://localhost:8080/swagger/index.html")
	log.Printf("📁 Database: PostgreSQL")

	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatal(err)
	}
}
