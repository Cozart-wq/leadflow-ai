package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/leadflow-ai/leadflow-ai/internal/middleware"
	"github.com/leadflow-ai/leadflow-ai/internal/services"
)

type TaskHandler struct {
	tasks    *services.TaskService
	pipeline *services.PipelineService
	log      *zap.Logger
}

func NewTaskHandler(tasks *services.TaskService, pipeline *services.PipelineService, log *zap.Logger) *TaskHandler {
	return &TaskHandler{tasks: tasks, pipeline: pipeline, log: log}
}

func (h *TaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := authenticatedUserID(w, r)
	if !ok {
		return
	}

	var req createTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	task, err := h.tasks.Create(r.Context(), userID, req.Query)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	h.pipeline.RunAsync(task.ID, task.Query)

	writeJSON(w, http.StatusCreated, task)
}

func (h *TaskHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := authenticatedUserID(w, r)
	if !ok {
		return
	}

	limit, offset := parsePagination(r)
	tasks, err := h.tasks.List(r.Context(), userID, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list tasks")
		return
	}
	writeJSON(w, http.StatusOK, tasks)
}

func (h *TaskHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID, ok := authenticatedUserID(w, r)
	if !ok {
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid task id")
		return
	}
	task, err := h.tasks.Get(r.Context(), userID, id)
	if err != nil {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}
	writeJSON(w, http.StatusOK, task)
}

type LeadHandler struct {
	leads *services.LeadService
	log   *zap.Logger
}

func NewLeadHandler(leads *services.LeadService, log *zap.Logger) *LeadHandler {
	return &LeadHandler{leads: leads, log: log}
}

func (h *LeadHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := authenticatedUserID(w, r)
	if !ok {
		return
	}

	limit, offset := parsePagination(r)

	var taskID *uuid.UUID
	if raw := r.URL.Query().Get("task_id"); raw != "" {
		parsed, err := uuid.Parse(raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid task_id")
			return
		}
		taskID = &parsed
	}

	leads, err := h.leads.List(r.Context(), userID, taskID, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list leads")
		return
	}
	writeJSON(w, http.StatusOK, leads)
}

func (h *LeadHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID, ok := authenticatedUserID(w, r)
	if !ok {
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid lead id")
		return
	}
	lead, err := h.leads.Get(r.Context(), userID, id)
	if err != nil {
		writeError(w, http.StatusNotFound, "lead not found")
		return
	}
	writeJSON(w, http.StatusOK, lead)
}

func (h *LeadHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := authenticatedUserID(w, r)
	if !ok {
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid lead id")
		return
	}
	if err := h.leads.Delete(r.Context(), userID, id); err != nil {
		writeError(w, http.StatusNotFound, "lead not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// authenticatedUserID читает claims, положенные middleware.Auth, и
// возвращает id пользователя. Второе значение false означает, что
// middleware.Auth не была подключена к этому маршруту — в таком случае
// обработчик сразу отвечает 401 и сигнализирует вызывающему коду не
// продолжать обработку запроса.
func authenticatedUserID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return uuid.UUID{}, false
	}
	return claims.UserID, true
}

func parsePagination(r *http.Request) (limit, offset int) {
	limit = 50
	offset = 0
	if v := r.URL.Query().Get("limit"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			limit = parsed
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			offset = parsed
		}
	}
	return limit, offset
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{Error: message})
}
