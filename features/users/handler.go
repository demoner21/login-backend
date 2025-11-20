package users

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type RegisterRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type Response struct {
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func Routes(s *Service) (string, func(r chi.Router)) {
	return "/users", func(r chi.Router) {
		// Registro
		r.Post("/register", func(w http.ResponseWriter, r *http.Request) {
			var body RegisterRequest

			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, "dados inválidos", http.StatusBadRequest)
				return
			}

			if err := s.Register(body.Name, body.Email, body.Password); err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(Response{Error: err.Error()})
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(Response{Message: "usuário criado com sucesso"})
		})

		// Login
		r.Post("/login", func(w http.ResponseWriter, r *http.Request) {
			var body LoginRequest

			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, "dados inválidos", http.StatusBadRequest)
				return
			}

			user, err := s.Login(body.Email, body.Password)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(Response{Error: err.Error()})
				return
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(Response{Data: user})
		})

		// Listagem
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			users, err := s.ListUsers()
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(Response{Error: "erro ao listar usuários"})
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(Response{Data: users})
		})
	}
}
