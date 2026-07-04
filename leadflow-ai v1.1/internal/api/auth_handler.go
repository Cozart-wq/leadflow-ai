package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"go.uber.org/zap"

	"github.com/leadflow-ai/leadflow-ai/internal/middleware"
	"github.com/leadflow-ai/leadflow-ai/internal/models"
	"github.com/leadflow-ai/leadflow-ai/internal/services"
)

type AuthHandler struct {
	auth *services.AuthService
	log  *zap.Logger
}

func NewAuthHandler(auth *services.AuthService, log *zap.Logger) *AuthHandler {
	return &AuthHandler{auth: auth, log: log}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, token, err := h.auth.Register(r.Context(), req.Email, req.Password, req.Name)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrEmailTaken):
			writeError(w, http.StatusConflict, "email already registered")
		default:
			writeError(w, http.StatusBadRequest, err.Error())
		}
		return
	}

	writeJSON(w, http.StatusCreated, toAuthResponse(user, token))
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, token, err := h.auth.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	writeJSON(w, http.StatusOK, toAuthResponse(user, token))
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	user, err := h.auth.Profile(r.Context(), claims.UserID)
	if err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	writeJSON(w, http.StatusOK, userResponse{
		ID:        user.ID.String(),
		Email:     user.Email,
		Name:      user.Name,
		Role:      user.Role,
		CreatedAt: user.CreatedAt,
	})
}

func toAuthResponse(user *models.User, token string) authResponse {
	return authResponse{
		Token: token,
		User: userResponse{
			ID:        user.ID.String(),
			Email:     user.Email,
			Name:      user.Name,
			Role:      user.Role,
			CreatedAt: user.CreatedAt,
		},
	}
}
