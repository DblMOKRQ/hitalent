package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"go.uber.org/zap"
	"hitalent/internal/transport/http/constants"
	"hitalent/internal/transport/http/dto"
	"hitalent/internal/transport/http/middleware"
	"net/http"
	"strconv"

	"hitalent/internal/domain"
	"hitalent/internal/transport/http/response"
)

type departmentService interface {
	Create(ctx context.Context, dept *domain.Department) error
	GetByIDWithTree(ctx context.Context, id int, depth int, includeEmployees bool) (*domain.Department, error)
	Update(ctx context.Context, id int, newName *string, newParentID *int) (*domain.Department, error)
	Delete(ctx context.Context, id int, mode string, reassignToID *int) error
}

type DepartmentHandler struct {
	service departmentService
}

func NewDepartmentHandler(service departmentService) *DepartmentHandler {
	return &DepartmentHandler{service: service}
}

// handleError вспомогательный метод для маппинга сервисных ошибок в HTTP ответы
func (h *DepartmentHandler) handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		response.SendError(w, http.StatusNotFound, constants.CodeNotFound, constants.MsgNotFoundDepart)
	case errors.Is(err, domain.ErrInvalidInput):
		response.SendError(w, http.StatusBadRequest, constants.CodeInvalidInput, constants.MsgInvalidInput)
	case errors.Is(err, domain.ErrConflict) || errors.Is(err, domain.ErrCyclicTree):
		response.SendError(w, http.StatusConflict, constants.CodeConflict, constants.MsgConflict)
	default:
		response.SendError(w, http.StatusInternalServerError, constants.CodeInternalError, constants.MsgInternalError)
	}
}

// POST /departments/
func (h *DepartmentHandler) Create(w http.ResponseWriter, r *http.Request) {
	log := middleware.GetLogger(r.Context())

	var req dto.CreateDepartmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Warn("Invalid request body", zap.Error(err))
		response.SendError(w, http.StatusBadRequest, constants.CodeInvalidJson, constants.MsgInvalidJson)
		return
	}

	dept := &domain.Department{
		Name:     req.Name,
		ParentID: req.ParentID,
	}

	if err := h.service.Create(r.Context(), dept); err != nil {
		log.Warn("Failed to create department", zap.Error(err))
		h.handleError(w, err)
		return
	}

	response.SendJSON(w, http.StatusCreated, dept)
}

// GET /departments/{id}
func (h *DepartmentHandler) Get(w http.ResponseWriter, r *http.Request) {
	log := middleware.GetLogger(r.Context())
	id, err := strconv.Atoi(r.PathValue(constants.ID))
	if err != nil {
		log.Warn("Invalid department ID", zap.Error(err))
		response.SendError(w, http.StatusBadRequest, constants.CodeInvalidID, constants.MsgInvalidID)
		return
	}

	depth := 1
	if depthQuery := r.URL.Query().Get("depth"); depthQuery != "" {
		if d, err := strconv.Atoi(depthQuery); err == nil {
			depth = d
		}
	}

	includeEmployees := true
	if incEmpQuery := r.URL.Query().Get("include_employees"); incEmpQuery == "false" {
		includeEmployees = false
	}

	dept, err := h.service.GetByIDWithTree(r.Context(), id, depth, includeEmployees)
	if err != nil {
		log.Warn("Failed to get department", zap.Error(err))
		h.handleError(w, err)
		return
	}

	resp := dto.MapToDepartmentResponse(dept)
	response.SendJSON(w, http.StatusOK, resp)
}

// PATCH /departments/{id}
func (h *DepartmentHandler) Update(w http.ResponseWriter, r *http.Request) {
	log := middleware.GetLogger(r.Context())
	id, err := strconv.Atoi(r.PathValue(constants.ID))
	if err != nil {
		log.Warn("Invalid department ID", zap.Error(err))
		response.SendError(w, http.StatusBadRequest, constants.CodeInvalidID, constants.MsgInvalidID)
		return
	}

	var req dto.UpdateDepartmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.SendError(w, http.StatusBadRequest, constants.CodeInvalidJson, constants.MsgInvalidJson)
		return
	}

	dept, err := h.service.Update(r.Context(), id, req.Name, req.ParentID)
	if err != nil {
		log.Warn("Failed to update department", zap.Error(err))
		h.handleError(w, err)
		return
	}

	response.SendJSON(w, http.StatusOK, dept)
}

// DELETE /departments/{id}
func (h *DepartmentHandler) Delete(w http.ResponseWriter, r *http.Request) {
	log := middleware.GetLogger(r.Context())
	id, err := strconv.Atoi(r.PathValue(constants.ID))
	if err != nil {
		log.Warn("Invalid department ID", zap.Error(err))
		response.SendError(w, http.StatusBadRequest, constants.CodeInvalidID, constants.MsgInvalidID)
		return
	}

	mode := r.URL.Query().Get("mode")
	if mode == "" {
		mode = "cascade"
	}

	var reassignToID *int
	if rQuery := r.URL.Query().Get("reassign_to_department_id"); rQuery != "" {
		if rID, err := strconv.Atoi(rQuery); err == nil {
			reassignToID = &rID
		}
	}

	if err := h.service.Delete(r.Context(), id, mode, reassignToID); err != nil {
		log.Warn("Failed to delete department", zap.Error(err))
		h.handleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
