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
)

func main() {
	cfg := config.Load()

	db, err := database.NewDuckDB(cfg.DuckDBPath)
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

	// Registrar feature de autenticação
	authRepo := auth.NewRepository(db)
	authService := auth.NewService(authRepo, auth.Config{
		JWTSecret:     jwtSecret,
		AccessExpiry:  5 * time.Minute,  // 24 horas
		RefreshExpiry: 15 * time.Minute, // 7 dias
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
	log.Printf("📁 Database: %s", cfg.DuckDBPath)

	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatal(err)
	}
}
