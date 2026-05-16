package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"go.uber.org/zap"
	"hitalent/internal/domain"
)

const (
	defaultDepth = 1
	maxDepth     = 5
)

type DepartmentRepository interface {
	Create(ctx context.Context, dept *domain.Department) error
	GetByID(ctx context.Context, id int) (*domain.Department, error)
	GetWithTree(ctx context.Context, id int, depth int, includeEmployees bool) (*domain.Department, error)
	Update(ctx context.Context, dept *domain.Department) error
	Delete(ctx context.Context, id int) error
	ReassignEmployees(ctx context.Context, fromDeptID int, toDeptID int) error
	CheckDuplicateName(ctx context.Context, name string, parentID *int) (bool, error)
	IsDescendant(ctx context.Context, targetDeptID int, potentialParentID int) (bool, error)
}

type DepartmentService struct {
	departmentRepo DepartmentRepository
	log            *zap.Logger
}

func NewDepartmentService(departmentRepo DepartmentRepository, log *zap.Logger) *DepartmentService {
	return &DepartmentService{
		departmentRepo: departmentRepo,
		log:            log.Named("DepartmentService"),
	}
}

func (s *DepartmentService) Create(ctx context.Context, dept *domain.Department) error {
	dept.Name = strings.TrimSpace(dept.Name)
	s.log.Debug("Create department", zap.String("Name", dept.Name))

	if len(dept.Name) == 0 || len(dept.Name) > 200 {
		s.log.Warn("Department name invalid", zap.String("Name", dept.Name))
		return domain.ErrInvalidInput
	}

	if dept.ParentID != nil {
		_, err := s.departmentRepo.GetByID(ctx, *dept.ParentID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				s.log.Warn("Parent department not found", zap.Int("ID", dept.ID))
				return domain.ErrNotFound
			}
			return fmt.Errorf("failed to check department by id (%d): %w", dept.ID, err)
		}
	}

	isDuplicate, err := s.departmentRepo.CheckDuplicateName(ctx, dept.Name, dept.ParentID)
	if err != nil {
		return fmt.Errorf("failed to check department by name (%s): %w", dept.Name, err)
	}
	if isDuplicate {
		s.log.Warn("Department already exists", zap.String("Name", dept.Name))
		return domain.ErrConflict
	}

	if err := s.departmentRepo.Create(ctx, dept); err != nil {
		return fmt.Errorf("failed to create department: %w", err)
	}
	s.log.Debug("Create department success", zap.String("Name", dept.Name))
	return nil
}

func (s *DepartmentService) GetByIDWithTree(ctx context.Context, id int, depth int, includeEmployees bool) (*domain.Department, error) {
	s.log.Debug("Get department with tree", zap.Int("ID", id), zap.Int("Depth", depth))

	if depth < defaultDepth {
		depth = defaultDepth
	}
	if depth > maxDepth {
		depth = maxDepth
	}

	dept, err := s.departmentRepo.GetWithTree(ctx, id, depth, includeEmployees)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			s.log.Warn("Department not found", zap.Int("ID", id))
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get department with tree (%d): %w", id, err)
	}

	s.log.Debug("Get department with tree success", zap.Int("ID", id))
	return dept, nil
}

func (s *DepartmentService) Update(ctx context.Context, id int, newName *string, newParentID *int) (*domain.Department, error) {
	s.log.Debug("Update department", zap.Int("ID", id))

	dept, err := s.departmentRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get department by id (%d): %w", id, err)
	}

	nameToSave := dept.Name

	if newName != nil {
		trimmed := strings.TrimSpace(*newName)
		if len(trimmed) == 0 || len(trimmed) > 200 {
			s.log.Warn("Department new name invalid", zap.String("Name", trimmed))
			return nil, domain.ErrInvalidInput
		}
		nameToSave = trimmed
		dept.Name = nameToSave
	}

	if err := s.validateAndSetParent(ctx, dept, newParentID); err != nil {
		return nil, err
	}

	isDuplicate, err := s.departmentRepo.CheckDuplicateName(ctx, nameToSave, dept.ParentID)
	if err != nil {
		return nil, fmt.Errorf("failed to check duplicate name during update (%s): %w", nameToSave, err)
	}
	if isDuplicate && (newName != nil || newParentID != nil) {
		s.log.Warn("Department name conflict during update", zap.String("Name", nameToSave))
		return nil, domain.ErrConflict
	}

	if err := s.departmentRepo.Update(ctx, dept); err != nil {
		return nil, fmt.Errorf("failed to update department: %w", err)
	}

	s.log.Debug("Update department success", zap.Int("ID", id))
	return dept, nil
}

func (s *DepartmentService) Delete(ctx context.Context, id int, mode string, reassignToID *int) error {
	s.log.Debug("Delete department", zap.Int("ID", id), zap.String("Mode", mode))

	_, err := s.departmentRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			s.log.Warn("Department to delete not found", zap.Int("ID", id))
			return domain.ErrNotFound
		}
		return fmt.Errorf("failed to get department to delete (%d): %w", id, err)
	}

	if mode == "reassign" {
		if reassignToID == nil {
			s.log.Warn("Reassign mode requires reassignToID", zap.Int("ID", id))
			return domain.ErrInvalidInput
		}

		_, err := s.departmentRepo.GetByID(ctx, *reassignToID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				s.log.Warn("Target department for reassign not found", zap.Int("ReassignToID", *reassignToID))
				return domain.ErrNotFound
			}
			return fmt.Errorf("failed to get target department for reassign (%d): %w", *reassignToID, err)
		}

		err = s.departmentRepo.ReassignEmployees(ctx, id, *reassignToID)
		if err != nil {
			return fmt.Errorf("failed to reassign employees: %w", err)
		}
		s.log.Debug("Employees reassigned successfully", zap.Int("FromDept", id), zap.Int("ToDept", *reassignToID))
	} else if mode != "cascade" {
		s.log.Warn("Invalid delete mode provided", zap.String("Mode", mode))
		return domain.ErrInvalidInput
	}

	if err := s.departmentRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete department: %w", err)
	}

	s.log.Debug("Delete department success", zap.Int("ID", id))
	return nil
}

// validateAndSetParent вынесена для снижения цикломатической сложности метода Update.
// Проверяет существование нового родителя и защищает от создания циклов в дереве.
func (s *DepartmentService) validateAndSetParent(ctx context.Context, dept *domain.Department, newParentID *int) error {
	if newParentID == nil {
		return nil
	}

	if *newParentID == dept.ID {
		s.log.Warn("Cannot set department as its own parent", zap.Int("ID", dept.ID))
		return domain.ErrConflict
	}

	_, err := s.departmentRepo.GetByID(ctx, *newParentID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			s.log.Warn("New parent department not found", zap.Int("ParentID", *newParentID))
			return domain.ErrNotFound
		}
		return fmt.Errorf("failed to check new parent department (%d): %w", *newParentID, err)
	}

	isDesc, err := s.departmentRepo.IsDescendant(ctx, dept.ID, *newParentID)
	if err != nil {
		return fmt.Errorf("failed to check descendant relationship: %w", err)
	}

	if isDesc {
		s.log.Warn("Cyclic tree detected", zap.Int("TargetID", dept.ID), zap.Int("PotentialParentID", *newParentID))
		return domain.ErrCyclicTree
	}

	dept.ParentID = newParentID
	return nil
}
