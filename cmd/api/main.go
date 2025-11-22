package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"loginbackend/config"
	"loginbackend/features/auth"
	"loginbackend/features/shared/models"
	"loginbackend/features/users"
	"loginbackend/internal/database"
	httpPlatform "loginbackend/internal/http"
	"loginbackend/pkg/utils"

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

	seedSuperAdmin(db, cfg)

	r := httpPlatform.NewRouter(cfg, redisClient)

	// Swagger
	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("http://localhost:8080/swagger/doc.json"),
	))

	// Registrar feature de autenticação
	authRepo := auth.NewRepository(db)
	authService := auth.NewService(authRepo, redisClient, auth.Config{
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

	usersPath, usersRoutes := users.Routes(usersHandler, cfg.JWTSecret, redisClient)
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

func seedSuperAdmin(db *sql.DB, cfg *config.Config) {
	// Se não tiver configuração no .env, ignora
	if cfg.AdminEmail == "" || cfg.AdminPassword == "" {
		log.Println("ℹ️ Variáveis ADMIN_EMAIL/PASSWORD não definidas. Pulando criação de Super Admin.")
		return
	}

	repo := users.NewRepository(db)

	// Verifica se já existe
	exists, err := repo.EmailExists(cfg.AdminEmail)
	if err != nil {
		log.Printf("⚠️ Erro ao verificar existência do admin: %v", err)
		return
	}
	if exists {
		log.Println("ℹ️ Super Admin já existe no banco.")
		return
	}

	log.Println("🔨 Criando Super Admin automático...")

	// Hash da senha
	hash, err := utils.HashPassword(cfg.AdminPassword)
	if err != nil {
		log.Printf("❌ Erro ao gerar hash do admin: %v", err)
		return
	}

	// Cria o modelo do Admin (RoleID 1 = SUPER_ADMIN conforme sua migration)
	adminUser := models.User{
		ID:           utils.GenerateSnowflakeID(),
		Name:         "Super Admin",
		Email:        cfg.AdminEmail,
		PasswordHash: hash,
		RoleID:       1, // SUPER_ADMIN
		IsActive:     true,
	}

	// Salva no banco
	if err := repo.Create(adminUser); err != nil {
		log.Printf("❌ Erro ao salvar Super Admin: %v", err)
		return
	}

	log.Printf("✅ Super Admin criado com sucesso: %s", cfg.AdminEmail)
}
