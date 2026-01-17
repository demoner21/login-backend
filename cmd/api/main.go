package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"loginbackend/config"
	"loginbackend/features/auth"
	"loginbackend/features/shared/models"
	"loginbackend/features/tasks"
	"loginbackend/features/users"
	"loginbackend/internal/database"
	httpPlatform "loginbackend/internal/http"
	ws "loginbackend/internal/websocket"
	"loginbackend/pkg/utils"

	_ "loginbackend/docs"

	httpSwagger "github.com/swaggo/http-swagger"
)

// @title Login Backend API
// @version 1.0
// @description API de autenticação, usuários e tasks com suporte a WebSocket
// @host localhost:8080
// @BasePath /

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	cfg := config.Load()

	// Inicializar Snowflake ID
	if err := utils.InitSnowflake(1); err != nil {
		log.Fatal("Erro ao inicializar Snowflake:", err)
	}

	// Conexão com PostgreSQL
	db, err := database.NewPostgres(cfg.GetConnectionString())
	if err != nil {
		log.Fatal(err)
	}

	// Conexão com Redis
	redisClient, err := database.NewRedis(
		cfg.RedisHost,
		cfg.RedisPort,
		cfg.RedisPassword,
	)
	if err != nil {
		log.Fatal("Erro ao conectar no Redis:", err)
	}

	// Rodar migrations
	if err := database.RunMigrations(db, "./migrations"); err != nil {
		log.Fatal(err)
	}

	seedSuperAdmin(db, cfg)

	// Inicializar Router
	r := httpPlatform.NewRouter(cfg, redisClient)

	// Swagger
	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("http://localhost:8080/swagger/doc.json"),
	))

	// ======================================================
	// WebSocket Hub
	// ======================================================
	hub := ws.NewHub(redisClient)
	go hub.Run(context.Background())

	// ======================================================
	// Auth Feature
	// ======================================================
	authRepo := auth.NewRepository(db)
	authService := auth.NewService(authRepo, redisClient, auth.Config{
		JWTSecret:     cfg.JWTSecret,
		AccessExpiry:  5 * time.Minute,
		RefreshExpiry: 15 * time.Minute,
	})
	authHandler := auth.NewHandler(authService)

	authPath, authRoutes := auth.Routes(authHandler, redisClient)
	r.Route(authPath, authRoutes)

	// ======================================================
	// Users Feature
	// ======================================================
	usersRepo := users.NewRepository(db)
	usersService := users.NewService(usersRepo)
	usersHandler := users.NewHandler(usersService)

	usersPath, usersRoutes := users.Routes(
		usersHandler,
		cfg.JWTSecret,
		redisClient,
	)
	r.Route(usersPath, usersRoutes)

	// ======================================================
	// Tasks Feature (HTTP + WebSocket)
	// ======================================================
	tasksRepo := tasks.NewRepository(db)
	tasksService := tasks.NewService(tasksRepo)
	tasksHandler := tasks.NewHandler(tasksService, hub)

	tasksPath, tasksRoutes := tasks.Routes(
		tasksHandler,
		cfg.JWTSecret,
		redisClient,
	)
	r.Route(tasksPath, tasksRoutes)

	// ======================================================
	// Health Check
	// ======================================================
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		if err := db.Ping(); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"status": "db_unhealthy",
			})
			return
		}

		if err := redisClient.Ping(r.Context()).Err(); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"status": "redis_unhealthy",
			})
			return
		}

		_ = json.NewEncoder(w).Encode(map[string]string{
			"status": "healthy",
		})
	})

	log.Println("🚀 API rodando em http://localhost:8080")
	log.Println("📚 Swagger disponível em http://localhost:8080/swagger/index.html")
	log.Println("📁 Stack: PostgreSQL + Redis + WebSocket")

	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatal(err)
	}
}

func seedSuperAdmin(db *sql.DB, cfg *config.Config) {
	if cfg.AdminEmail == "" || cfg.AdminPassword == "" {
		log.Println("ℹ️ ADMIN_EMAIL/PASSWORD não definidos. Pulando Super Admin.")
		return
	}

	repo := users.NewRepository(db)

	exists, err := repo.EmailExists(cfg.AdminEmail)
	if err != nil {
		log.Printf("⚠️ Erro ao verificar admin: %v", err)
		return
	}
	if exists {
		log.Println("ℹ️ Super Admin já existe.")
		return
	}

	log.Println("🔨 Criando Super Admin automático...")

	hash, err := utils.HashPassword(cfg.AdminPassword)
	if err != nil {
		log.Printf("❌ Erro ao gerar hash: %v", err)
		return
	}

	adminUser := models.User{
		ID:           utils.GenerateSnowflakeID(),
		Name:         "Super Admin",
		Email:        cfg.AdminEmail,
		PasswordHash: hash,
		RoleID:       1,
		IsActive:     true,
	}

	if err := repo.Create(adminUser); err != nil {
		log.Printf("❌ Erro ao salvar Super Admin: %v", err)
		return
	}

	log.Printf("✅ Super Admin criado: %s", cfg.AdminEmail)
}
