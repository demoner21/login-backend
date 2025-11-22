package users

import (
	"encoding/json"
	"loginbackend/config"
	"loginbackend/pkg/uploader"
	"net/http"

	"github.com/go-chi/chi/v5"
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

func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req CreateUserRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "dados inválidos", http.StatusBadRequest)
		return
	}

	user, err := h.service.Create(req)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Response{Error: err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(Response{
		Message: "usuário criado com sucesso",
		Data:    user,
	})
}

func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.service.List()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Response{Error: "erro ao listar usuários"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Response{Data: users})
}

func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {
	// ✅ CORREÇÃO: ID agora é string (Snowflake ID)
	userID := chi.URLParam(r, "id")

	user, err := h.service.GetByID(userID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(Response{Error: err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Response{Data: user})
}

func (h *Handler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	// ✅ CORREÇÃO: ID agora é string (Snowflake ID)
	userID := chi.URLParam(r, "id")

	var req UpdateUserRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "dados inválidos", http.StatusBadRequest)
		return
	}

	user, err := h.service.Update(userID, req)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Response{Error: err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Response{
		Message: "usuário atualizado com sucesso",
		Data:    user,
	})
}

func (h *Handler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	// ✅ CORREÇÃO: ID agora é string (Snowflake ID)
	userID := chi.URLParam(r, "id")

	if err := h.service.Delete(userID); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Response{Error: err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Response{Message: "usuário deletado com sucesso"})
}

func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	// Pegar ID da URL
	userID := chi.URLParam(r, "id")

	var req ChangePasswordRequest

	// Decodificar JSON
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON inválido", http.StatusBadRequest)
		return
	}

	// Chamar serviço
	// O erro "bool and nil" acontecia aqui se o Service retornasse bool.
	// Agora que o Service retorna error, a comparação (err != nil) funciona.
	if err := h.service.ChangePassword(userID, req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest) // Ou 401/403 dependendo do erro, mas 400 serve
		json.NewEncoder(w).Encode(Response{Error: err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(Response{Message: "Senha alterada com sucesso"})
}

// UploadAvatar
// @Summary Atualizar avatar do usuário
// @Description Recebe uma imagem via multipart/form-data e atualiza o perfil
// @Tags users
// @Accept multipart/form-data
// @Produce json
// @Param id path string true "User ID"
// @Param avatar formData file true "Arquivo de imagem"
// @Success 200 {object} Response
// @Router /users/{id}/avatar [post]
func (h *Handler) UploadAvatar(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")

	// 1. Limitar tamanho do upload (ex: 5MB)
	r.ParseMultipartForm(5 << 20)

	// 2. Pegar o arquivo do form key "avatar"
	file, header, err := r.FormFile("avatar")
	if err != nil {
		http.Error(w, "Erro ao ler arquivo: avatar é obrigatório", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// 3. Processar Upload (usando nosso helper)
	// Precisamos injetar a config no Handler ou pegar via helper.
	// Assumindo que você tem acesso a cfg via h.service ou global, ou passe cfg na inicialização do Handler.
	// Para simplificar aqui, vamos supor que o Service tem a config:
	cfg := config.Load() // Ou injetado

	avatarURL, err := uploader.UploadFile(file, header, cfg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 4. Atualizar registro no Banco (Service)
	updatedUser, err := h.service.UpdateAvatar(userID, avatarURL)
	if err != nil {
		http.Error(w, "Erro ao atualizar perfil no banco", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Response{
		Message: "Avatar atualizado com sucesso",
		Data:    updatedUser,
	})
}
