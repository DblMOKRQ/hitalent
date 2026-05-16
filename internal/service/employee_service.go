package service

import (
	"context"
	"errors"
	"fmt"
	"go.uber.org/zap"
	"hitalent/internal/domain"
	"strings"
)

type EmployeeRepository interface {
	Create(ctx context.Context, emp *domain.Employee) error
}

type departmentProvider interface {
	GetByID(ctx context.Context, id int) (*domain.Department, error)
}

type EmployeeService struct {
	employeeRepo EmployeeRepository
	deptProvider departmentProvider
	log          *zap.Logger
}

func NewEmployeeService(employeeRepo EmployeeRepository, deptProvider departmentProvider, log *zap.Logger) *EmployeeService {
	return &EmployeeService{
		employeeRepo: employeeRepo,
		deptProvider: deptProvider,
		log:          log.Named("EmployeeService"),
	}
}

func (s *EmployeeService) Create(ctx context.Context, emp *domain.Employee) error {
	emp.FullName = strings.TrimSpace(emp.FullName)
	emp.Position = strings.TrimSpace(emp.Position)
	s.log.Debug("Create Employee", zap.String("Full Name", emp.FullName), zap.String("Position", emp.Position))
	if len(emp.FullName) == 0 || len(emp.FullName) > 200 {
		s.log.Warn("Full Name is invalid", zap.String("Full Name", emp.FullName))
		return domain.ErrInvalidInput
	}
	if len(emp.Position) == 0 || len(emp.Position) > 200 {
		s.log.Warn("Position is invalid", zap.String("Position", emp.Position))
		return domain.ErrInvalidInput
	}

	_, err := s.deptProvider.GetByID(ctx, emp.DepartmentID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			s.log.Warn("Department not found", zap.Int("DepartmentID", emp.DepartmentID))
			return domain.ErrNotFound
		}
		return fmt.Errorf("failed to check department by id (%d): %w", emp.DepartmentID, err)
	}

	if err = s.employeeRepo.Create(ctx, emp); err != nil {
		return fmt.Errorf("failed to create employee: %w", err)
	}

	s.log.Debug("Employee create success", zap.String("Full Name", emp.FullName), zap.String("Position", emp.Position))
	return nil
}
