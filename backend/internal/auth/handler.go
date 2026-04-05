package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/mail"
	"strings"
)

type Handler struct {
	service *Service
	jwt     *JWT
}

func NewHandler(service *Service, jwt *JWT) *Handler {
	return &Handler{
		service: service,
		jwt:     jwt,
	}
}

type registerRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_BODY", "corpo da requisição inválido")
		return
	}

	req.normalize()

	if err := req.validate(); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	if err := h.service.Register(r.Context(), req.Name, req.Email, req.Password); err != nil {
		writeError(w, http.StatusInternalServerError, "REGISTER_FAILED", "não foi possível criar o usuário")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"message": "user created successfully",
	})
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_BODY", "corpo da requisição inválido")
		return
	}

	req.normalize()

	if err := req.validate(); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	userID, err := h.service.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "credenciais inválidas")
		return
	}

	token, err := h.jwt.Generate(userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "TOKEN_ERROR", "erro ao gerar token")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"token": token,
	})
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "usuário não autenticado")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"user": map[string]string{
			"id": userID,
		},
	})
}

func (r *registerRequest) normalize() {
	r.Name = strings.TrimSpace(r.Name)
	r.Email = strings.ToLower(strings.TrimSpace(r.Email))
	r.Password = strings.TrimSpace(r.Password)
}

func (r *registerRequest) validate() error {
	switch {
	case r.Name == "":
		return errors.New("nome é obrigatório")
	case len([]rune(r.Name)) < 2:
		return errors.New("nome deve ter pelo menos 2 caracteres")
	case r.Email == "":
		return errors.New("email é obrigatório")
	case !isValidEmail(r.Email):
		return errors.New("email inválido")
	case r.Password == "":
		return errors.New("senha é obrigatória")
	case len(r.Password) < 6:
		return errors.New("senha deve ter pelo menos 6 caracteres")
	default:
		return nil
	}
}

func (r *loginRequest) normalize() {
	r.Email = strings.ToLower(strings.TrimSpace(r.Email))
	r.Password = strings.TrimSpace(r.Password)
}

func (r *loginRequest) validate() error {
	switch {
	case r.Email == "":
		return errors.New("email é obrigatório")
	case !isValidEmail(r.Email):
		return errors.New("email inválido")
	case r.Password == "":
		return errors.New("senha é obrigatória")
	default:
		return nil
	}
}

func isValidEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
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