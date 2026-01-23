package acl

import (
	"encoding/json"
	"net/http"

	"loginbackend/internal/http/middleware"
	pkgacl "loginbackend/pkg/acl"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
)

// ============================================
// HANDLER
// ============================================

type Handler struct {
	service  *Service
	validate *validator.Validate
}

func NewHandler(service *Service) *Handler {
	return &Handler{
		service:  service,
		validate: validator.New(),
	}
}

type Response struct {
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// GrantACL
// @Summary Criar ou atualizar ACL
// @Description Concede permissões para um recurso
// @Tags acl
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body GrantACLRequest true "Dados da ACL"
// @Success 200 {object} Response
// @Router /acl [post]
func (h *Handler) GrantACL(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req GrantACLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Response{Error: "JSON inválido"})
		return
	}

	if err := h.validate.Struct(req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Response{Error: err.Error()})
		return
	}

	if err := h.service.GrantAccess(claims.UserID, req); err != nil {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(Response{Error: err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Response{Message: "Permissão concedida"})
}

// ShareResource
// @Summary Compartilhar recurso
// @Description Compartilha recurso com múltiplos usuários/times
// @Tags acl
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body ShareRequest true "Dados de compartilhamento"
// @Success 200 {object} Response
// @Router /share [post]
func (h *Handler) ShareResource(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req ShareRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Response{Error: "JSON inválido"})
		return
	}

	if err := h.validate.Struct(req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Response{Error: err.Error()})
		return
	}

	if err := h.service.Share(claims.UserID, req); err != nil {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(Response{Error: err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Response{Message: "Recurso compartilhado com sucesso"})
}

// GetACL
// @Summary Obter ACLs de um recurso
// @Description Lista todas as permissões de um recurso
// @Tags acl
// @Produce json
// @Security BearerAuth
// @Param resource_id path string true "Resource ID"
// @Param resource_type query string true "Resource Type" Enums(TASK, FARM_AREA, TEAM)
// @Success 200 {object} Response{data=[]ACL}
// @Router /acl/{resource_id} [get]
func (h *Handler) GetACL(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	resourceID := chi.URLParam(r, "resource_id")
	resourceType := pkgacl.ResourceType(r.URL.Query().Get("resource_type"))

	if resourceID == "" || resourceType == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Response{Error: "resource_id e resource_type obrigatórios"})
		return
	}

	acls, err := h.service.GetResourceACL(claims.UserID, resourceID, resourceType)
	if err != nil {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(Response{Error: err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Response{Data: acls})
}

// RevokeACL
// @Summary Revogar permissões
// @Description Remove permissões de um usuário/time
// @Tags acl
// @Produce json
// @Security BearerAuth
// @Param resource_id path string true "Resource ID"
// @Param resource_type query string true "Resource Type"
// @Param grantee_type query string true "Grantee Type" Enums(USER, TEAM, PUBLIC)
// @Param grantee_id query string false "Grantee ID (não enviar para PUBLIC)"
// @Success 200 {object} Response
// @Router /acl/{resource_id} [delete]
func (h *Handler) RevokeACL(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	resourceID := chi.URLParam(r, "resource_id")
	resourceType := pkgacl.ResourceType(r.URL.Query().Get("resource_type"))
	granteeType := pkgacl.GranteeType(r.URL.Query().Get("grantee_type"))
	granteeIDStr := r.URL.Query().Get("grantee_id")

	var granteeID *string
	if granteeIDStr != "" {
		granteeID = &granteeIDStr
	}

	if resourceID == "" || resourceType == "" || granteeType == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Response{Error: "parâmetros obrigatórios faltando"})
		return
	}

	if err := h.service.RevokeAccess(claims.UserID, resourceID, resourceType, granteeID, granteeType); err != nil {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(Response{Error: err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Response{Message: "Permissão revogada"})
}

// ListSharedWithMe
// @Summary Recursos compartilhados comigo
// @Description Lista recursos que foram compartilhados com o usuário
// @Tags acl
// @Produce json
// @Security BearerAuth
// @Param resource_type query string false "Filtrar por tipo"
// @Success 200 {object} Response{data=[]SharedResource}
// @Router /shared-with-me [get]
func (h *Handler) ListSharedWithMe(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var resourceType *pkgacl.ResourceType
	if rt := r.URL.Query().Get("resource_type"); rt != "" {
		t := pkgacl.ResourceType(rt)
		resourceType = &t
	}

	resources, err := h.service.ListSharedWithMe(claims.UserID, resourceType)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Response{Error: err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Response{Data: resources})
}

// ListSharedByMe
// @Summary Recursos compartilhados por mim
// @Description Lista recursos que o usuário compartilhou
// @Tags acl
// @Produce json
// @Security BearerAuth
// @Param resource_type query string false "Filtrar por tipo"
// @Success 200 {object} Response{data=[]SharedResource}
// @Router /shared-by-me [get]
func (h *Handler) ListSharedByMe(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var resourceType *pkgacl.ResourceType
	if rt := r.URL.Query().Get("resource_type"); rt != "" {
		t := pkgacl.ResourceType(rt)
		resourceType = &t
	}

	resources, err := h.service.ListSharedByMe(claims.UserID, resourceType)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Response{Error: err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Response{Data: resources})
}
