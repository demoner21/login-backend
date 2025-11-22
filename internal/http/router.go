package http

import (
	"loginbackend/config"
	"loginbackend/internal/http/middleware"
	"loginbackend/internal/http/ratelimit"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/go-chi/httprate"

	"github.com/redis/go-redis/v9"
)

func NewRouter(cfg *config.Config, redisClient *redis.Client) *chi.Mux {
	r := chi.NewRouter()

	origins := cfg.AllowedOrigins
	if len(origins) == 0 {
		origins = []string{"http://localhost:5173"}
	}

	// SECURITY HEADERS
	r.Use(middleware.SecurityHeaders)

	// Rate Limit GLOBAL (DDoS Protection)
	r.Use(httprate.Limit(
		100,
		1*time.Minute,
		httprate.WithKeyFuncs(httprate.KeyByIP),
		httprate.WithLimitCounter(ratelimit.NewRedisLimitCounter(redisClient, "global-rate")),
	))

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   origins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link", "X-Total-Count"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	workDir, _ := os.Getwd()
	uploadDir := filepath.Join(workDir, "uploads")

	// Rota para arquivos
	FileServer(r, "/uploads", http.Dir(uploadDir))

	return r
}

func FileServer(r chi.Router, path string, root http.FileSystem) {
	if strings.ContainsAny(path, "{}*") {
		panic("FileServer não permite parâmetros de URL")
	}

	fs := http.StripPrefix(path, http.FileServer(root))

	if path != "/" && path[len(path)-1] != '/' {
		r.Get(path, http.RedirectHandler(path+"/", 301).ServeHTTP)
		path += "/"
	}
	path += "*"

	r.Get(path, func(w http.ResponseWriter, r *http.Request) {
		fs.ServeHTTP(w, r)
	})
}
