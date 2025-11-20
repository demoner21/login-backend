package users

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func Routes(s *Service) (string, func(r chi.Router)) {
	return "/users", func(r chi.Router) {

		// Registro
		r.Post("/register", func(w http.ResponseWriter, r *http.Request) {
			var body struct {
				Name     string `json:"name"`
				Email    string `json:"email"`
				Password string `json:"password"`
			}

			json.NewDecoder(r.Body).Decode(&body)

			if err := s.Register(body.Name, body.Email, body.Password); err != nil {
				http.Error(w, err.Error(), 400)
				return
			}

			w.WriteHeader(201)
			w.Write([]byte(`{"message":"usuário criado com sucesso"}`))
		})

		// Login
		r.Post("/login", func(w http.ResponseWriter, r *http.Request) {
			var body struct {
				Email    string `json:"email"`
				Password string `json:"password"`
			}

			json.NewDecoder(r.Body).Decode(&body)

			user, err := s.Login(body.Email, body.Password)
			if err != nil {
				http.Error(w, err.Error(), 400)
				return
			}

			json.NewEncoder(w).Encode(user)
		})

		// Listagem
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			users, err := s.ListUsers()
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			json.NewEncoder(w).Encode(users)
		})
	}
}
