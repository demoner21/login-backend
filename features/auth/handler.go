package auth

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

type Response struct {
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// Login
// @Summary Login do usuário
// @Description Faz login do usuário e retorna tokens JWT
// @Tags auth
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Credenciais de login"
// @Success 200 {object} Response{data=LoginResponse}
// @Failure 401 {object} Response
// @Router /auth/login [post]
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "dados inválidos", http.StatusBadRequest)
		return
	}

	response, err := h.service.Login(req.Email, req.Password)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(Response{Error: err.Error()})
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    response.RefreshToken,
		Path:     "/auth",                            // O cookie só será enviado para rotas de autenticação
		Expires:  time.Now().Add(24 * time.Hour * 7), // Deve bater com a duração no Service (ex: 7 dias)
		HttpOnly: true,                               // Impossível de ler via JavaScript (XSS Protection)
		Secure:   true,                               // Só envia via HTTPS (Coloque 'false' se estiver testando local sem SSL)
		SameSite: http.SameSiteLaxMode,               // Proteção contra CSRF
	})

	response.RefreshToken = ""

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Response{Data: response})
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("refresh_token")
	if err != nil {
		if err == http.ErrNoCookie {
			http.Error(w, "refresh token não encontrado", http.StatusUnauthorized)
			return
		}
		http.Error(w, "erro ao ler cookie", http.StatusBadRequest)
		return
	}

	refreshTokenString := cookie.Value

	response, err := h.service.RefreshToken(refreshTokenString)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(Response{Error: err.Error()})
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    response.RefreshToken,
		Path:     "/auth",
		Expires:  time.Now().Add(24 * time.Hour * 7),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})

	response.RefreshToken = ""

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Response{Data: response})
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")

	authHeader := r.Header.Get("Authorization")
	tokenString := ""
	if authHeader != "" {
		parts := strings.Split(authHeader, " ")
		if len(parts) == 2 {
			tokenString = parts[1]
		}
	}

	if err := h.service.Logout(tokenString, userID); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Response{Error: "erro ao fazer logout"})
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/auth",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Response{Message: "logout realizado com sucesso"})
}
