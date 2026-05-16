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
	"time"

	"hitalent/internal/domain"
	"hitalent/internal/transport/http/response"
)

type EmployeeService interface {
	Create(ctx context.Context, emp *domain.Employee) error
}

type EmployeeHandler struct {
	service EmployeeService
}

func NewEmployeeHandler(service EmployeeService) *EmployeeHandler {
	return &EmployeeHandler{service: service}
}

func (h *EmployeeHandler) handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		response.SendError(w, http.StatusNotFound, constants.CodeNotFound, constants.MsgNotFoundEmpl)
	case errors.Is(err, domain.ErrInvalidInput):
		response.SendError(w, http.StatusBadRequest, constants.CodeInvalidInput, constants.MsgInvalidInput)
	default:
		response.SendError(w, http.StatusInternalServerError, constants.CodeInternalError, constants.MsgInternalError)
	}
}

// POST /departments/{id}/employees/
func (h *EmployeeHandler) Create(w http.ResponseWriter, r *http.Request) {
	log := middleware.GetLogger(r.Context())
	deptID, err := strconv.Atoi(r.PathValue(constants.ID))
	if err != nil {
		log.Warn("Invalid department ID", zap.Error(err))
		response.SendError(w, http.StatusBadRequest, constants.CodeInvalidID, constants.MsgInvalidID)
		return
	}

	var req dto.CreateEmployeeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Warn("Invalid request body", zap.Error(err))
		response.SendError(w, http.StatusBadRequest, constants.CodeInvalidJson, constants.MsgInvalidJson)
		return
	}

	emp := &domain.Employee{
		DepartmentID: deptID,
		FullName:     req.FullName,
		Position:     req.Position,
	}

	if req.HiredAt != nil && *req.HiredAt != "" {
		parsedDate, err := time.Parse(time.DateOnly, *req.HiredAt)
		if err != nil {
			log.Warn("Invalid date", zap.Error(err))
			response.SendError(w, http.StatusBadRequest, constants.CodeInvalidDate, constants.MsgInvalidDate)
			return
		}
		emp.HiredAt = &parsedDate
	}

	if err := h.service.Create(r.Context(), emp); err != nil {
		log.Warn("Failed to create employee", zap.Error(err))
		h.handleError(w, err)
		return
	}

	response.SendJSON(w, http.StatusCreated, emp)
}
