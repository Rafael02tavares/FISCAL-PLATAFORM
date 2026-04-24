package adminusers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/rafa/fiscal-platform/backend/internal/auth"
	"github.com/rafa/fiscal-platform/backend/internal/organizations"
)

type Handler struct {
	service    *Service
	orgService *organizations.Service
}

func NewHandler(service *Service, orgService *organizations.Service) *Handler {
	return &Handler{
		service:    service,
		orgService: orgService,
	}
}

type createUserRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

type updateRoleRequest struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	organizationID, ok := h.authorizeRequest(w, r)
	if !ok {
		return
	}

	items, err := h.service.ListByOrganization(r.Context(), organizationID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "LIST_USERS_FAILED", "nao foi possivel listar os usuarios da organizacao")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"items": items,
	})
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	organizationID, ok := h.authorizeRequest(w, r)
	if !ok {
		return
	}

	var req createUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_BODY", "corpo da requisicao invalido")
		return
	}

	if err := h.service.CreateOrAttachUser(r.Context(), CreateOrAttachUserParams{
		OrganizationID: organizationID,
		Name:           req.Name,
		Email:          req.Email,
		Password:       req.Password,
		Role:           req.Role,
	}); err != nil {
		switch {
		case errors.Is(err, ErrInvalidUserData):
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "nome, email e senha valida sao obrigatorios para novo usuario")
		case errors.Is(err, ErrInvalidRole):
			writeError(w, http.StatusBadRequest, "INVALID_ROLE", "papel de usuario invalido")
		case errors.Is(err, ErrUserAlreadyInOrg):
			writeError(w, http.StatusConflict, "USER_ALREADY_IN_ORG", "esse usuario ja faz parte da organizacao")
		default:
			writeError(w, http.StatusInternalServerError, "CREATE_USER_FAILED", "nao foi possivel cadastrar ou vincular o usuario")
		}
		return
	}

	items, _ := h.service.ListByOrganization(r.Context(), organizationID)
	writeJSON(w, http.StatusCreated, map[string]any{
		"message": "usuario salvo com sucesso",
		"items":   items,
	})
}

func (h *Handler) UpdateRole(w http.ResponseWriter, r *http.Request) {
	organizationID, ok := h.authorizeRequest(w, r)
	if !ok {
		return
	}

	var req updateRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_BODY", "corpo da requisicao invalido")
		return
	}

	if err := h.service.UpdateRole(r.Context(), organizationID, req.UserID, req.Role); err != nil {
		switch {
		case errors.Is(err, ErrInvalidUserData):
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "usuario invalido")
		case errors.Is(err, ErrInvalidRole):
			writeError(w, http.StatusBadRequest, "INVALID_ROLE", "papel de usuario invalido")
		case errors.Is(err, ErrUserMembershipMissing):
			writeError(w, http.StatusNotFound, "USER_MEMBERSHIP_NOT_FOUND", "vinculo do usuario com a organizacao nao encontrado")
		default:
			writeError(w, http.StatusInternalServerError, "UPDATE_ROLE_FAILED", "nao foi possivel atualizar o papel do usuario")
		}
		return
	}

	items, _ := h.service.ListByOrganization(r.Context(), organizationID)
	writeJSON(w, http.StatusOK, map[string]any{
		"message": "papel atualizado com sucesso",
		"items":   items,
	})
}

func (h *Handler) Remove(w http.ResponseWriter, r *http.Request) {
	organizationID, ok := h.authorizeRequest(w, r)
	if !ok {
		return
	}

	userID := strings.TrimSpace(r.URL.Query().Get("user_id"))
	if err := h.service.RemoveFromOrganization(r.Context(), organizationID, userID); err != nil {
		switch {
		case errors.Is(err, ErrInvalidUserData):
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "user_id e obrigatorio")
		case errors.Is(err, ErrUserMembershipMissing):
			writeError(w, http.StatusNotFound, "USER_MEMBERSHIP_NOT_FOUND", "vinculo do usuario com a organizacao nao encontrado")
		default:
			writeError(w, http.StatusInternalServerError, "REMOVE_USER_FAILED", "nao foi possivel remover o usuario da organizacao")
		}
		return
	}

	items, _ := h.service.ListByOrganization(r.Context(), organizationID)
	writeJSON(w, http.StatusOK, map[string]any{
		"message": "usuario removido da organizacao",
		"items":   items,
	})
}

func (h *Handler) authorizeRequest(w http.ResponseWriter, r *http.Request) (string, bool) {
	userID := auth.UserIDFromContext(r.Context())
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "usuario nao autenticado")
		return "", false
	}

	organizationID := strings.TrimSpace(r.Header.Get("X-Organization-ID"))
	if organizationID == "" {
		writeError(w, http.StatusBadRequest, "MISSING_ORGANIZATION_ID", "X-Organization-ID e obrigatorio")
		return "", false
	}

	allowed, err := h.orgService.UserBelongsToOrganization(r.Context(), userID, organizationID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "ORGANIZATION_VALIDATION_FAILED", "nao foi possivel validar acesso a organizacao")
		return "", false
	}

	if !allowed {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "usuario sem acesso a esta organizacao")
		return "", false
	}

	return organizationID, true
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}
