package dto

import (
	"hitalent/internal/domain"
	"time"
)

// --- Запросы (Requests) ---

type CreateDepartmentRequest struct {
	Name     string `json:"name"`
	ParentID *int   `json:"parent_id,omitempty"`
}

type UpdateDepartmentRequest struct {
	Name     *string `json:"name,omitempty"`      // Указатель, так как поле опционально
	ParentID *int    `json:"parent_id,omitempty"` // Указатель, так как может быть null
}

type CreateEmployeeRequest struct {
	FullName string  `json:"full_name"`
	Position string  `json:"position"`
	HiredAt  *string `json:"hired_at,omitempty"` // Можно принимать строку и парсить в time.Time
}

// --- Ответы (Responses) ---

// DepartmentResponse нужен, чтобы отдать данные в нужном формате с вложенными детьми и сотрудниками
type DepartmentResponse struct {
	ID        int                  `json:"id"`
	Name      string               `json:"name"`
	ParentID  *int                 `json:"parent_id"`
	CreatedAt time.Time            `json:"created_at"`
	Employees []domain.Employee    `json:"employees,omitempty"` // из models.go
	Children  []DepartmentResponse `json:"children,omitempty"`
}

// MapToDepartmentResponse рекурсивно конвертирует доменную модель в DTO для JSON ответа
func MapToDepartmentResponse(dept *domain.Department) DepartmentResponse {
	resp := DepartmentResponse{
		ID:        dept.ID,
		Name:      dept.Name,
		ParentID:  dept.ParentID,
		CreatedAt: dept.CreatedAt,
		Employees: dept.Employees,
	}

	for _, child := range dept.Children {
		resp.Children = append(resp.Children, MapToDepartmentResponse(&child))
	}

	return resp
}
