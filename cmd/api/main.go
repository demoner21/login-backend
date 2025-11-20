package main

import (
	"log"
	"net/http"

	"loginbackend/config"
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

	// registrar feature users
	repo := users.NewRepository(db)
	service := users.NewService(repo)
	path, routes := users.Routes(service)
	r.Route(path, routes)

	log.Println("🚀 API rodando em http://localhost:8080")
	http.ListenAndServe(":8080", r)
}
