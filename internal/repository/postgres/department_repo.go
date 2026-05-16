package repository

import (
	"context"
	"errors"
	"fmt"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"hitalent/internal/domain"
)

const (
	isDescendantQuery = `
		WITH RECURSIVE ancestors AS (
			SELECT id, parent_id FROM departments WHERE id = ?
			UNION
			SELECT d.id, d.parent_id FROM departments d
			INNER JOIN ancestors a ON d.id = a.parent_id
		)
		SELECT COUNT(*) FROM ancestors WHERE id = ?;
	`
)

type DepartmentRepository struct {
	db  *gorm.DB
	log *zap.Logger
}

func NewDepartmentRepository(db *gorm.DB, log *zap.Logger) *DepartmentRepository {
	return &DepartmentRepository{
		db:  db,
		log: log.Named("DepartmentRepository"),
	}
}

// Create создает новое подразделение
func (r *DepartmentRepository) Create(ctx context.Context, dept *domain.Department) error {
	r.log.Debug("Inserting new department", zap.String("name", dept.Name))

	if err := r.db.WithContext(ctx).Create(dept).Error; err != nil {
		return fmt.Errorf("failed to insert department: %w", err)
	}
	return nil
}

// GetByID получает подразделение без связей
func (r *DepartmentRepository) GetByID(ctx context.Context, id int) (*domain.Department, error) {
	r.log.Debug("Fetching department by ID", zap.Int("id", id))

	var dept domain.Department
	err := r.db.WithContext(ctx).First(&dept, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("failed to fetch department by id: %w", err)
	}
	return &dept, nil
}

// GetWithTree получает подразделение, его сотрудников и детей на заданную глубину
func (r *DepartmentRepository) GetWithTree(ctx context.Context, id int, depth int, includeEmployees bool) (*domain.Department, error) {
	r.log.Debug("Fetching department with tree", zap.Int("id", id), zap.Int("depth", depth), zap.Bool("includeEmployees", includeEmployees))

	var dept domain.Department
	query := r.db.WithContext(ctx)

	if includeEmployees {
		query = query.Preload("Employees", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at ASC")
		})
	}

	if depth > 1 {
		currentRelation := "Children"
		for i := 2; i <= depth; i++ {
			query = query.Preload(currentRelation)
			if includeEmployees {
				query = query.Preload(currentRelation+".Employees", func(db *gorm.DB) *gorm.DB {
					return db.Order("created_at ASC")
				})
			}
			currentRelation += ".Children"
		}
	}

	err := query.First(&dept, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("failed to fetch tree: %w", err)
	}

	return &dept, nil
}

// Update обновляет поля подразделения
func (r *DepartmentRepository) Update(ctx context.Context, dept *domain.Department) error {
	r.log.Debug("Updating department", zap.Int("id", dept.ID))

	if err := r.db.WithContext(ctx).Model(dept).Updates(dept).Error; err != nil {
		return fmt.Errorf("failed to update department: %w", err)
	}
	return nil
}

// Delete удаляет подразделение (каскадно)
func (r *DepartmentRepository) Delete(ctx context.Context, id int) error {
	r.log.Debug("Deleting department", zap.Int("id", id))

	if err := r.db.WithContext(ctx).Delete(&domain.Department{}, id).Error; err != nil {
		return fmt.Errorf("failed to delete department: %w", err)
	}
	return nil
}

// ReassignEmployees переводит сотрудников из одного департамента в другой
func (r *DepartmentRepository) ReassignEmployees(ctx context.Context, fromDeptID int, toDeptID int) error {
	r.log.Debug("Reassigning employees", zap.Int("fromDept", fromDeptID), zap.Int("toDept", toDeptID))

	err := r.db.WithContext(ctx).Model(&domain.Employee{}).
		Where("department_id = ?", fromDeptID).
		Update("department_id", toDeptID).Error

	if err != nil {
		return fmt.Errorf("failed to reassign employees: %w", err)
	}
	return nil
}

// CheckDuplicateName проверяет, есть ли уже такое имя в рамках одного родителя
func (r *DepartmentRepository) CheckDuplicateName(ctx context.Context, name string, parentID *int) (bool, error) {
	r.log.Debug("Checking duplicate name", zap.String("name", name), zap.Any("parentID", parentID))

	var count int64
	query := r.db.WithContext(ctx).Model(&domain.Department{}).Where("name = ?", name)

	if parentID == nil {
		query = query.Where("parent_id IS NULL")
	} else {
		query = query.Where("parent_id = ?", parentID)
	}

	if err := query.Count(&count).Error; err != nil {
		return false, fmt.Errorf("failed to count duplicate names: %w", err)
	}
	return count > 0, nil
}

// IsDescendant проверяет, является ли potentialParent потомком targetDept
func (r *DepartmentRepository) IsDescendant(ctx context.Context, targetDeptID int, potentialParentID int) (bool, error) {
	r.log.Debug("Checking descendant relationship", zap.Int("target", targetDeptID), zap.Int("potentialParent", potentialParentID))

	var count int64
	err := r.db.WithContext(ctx).Raw(isDescendantQuery, potentialParentID, targetDeptID).Scan(&count).Error
	if err != nil {
		return false, fmt.Errorf("failed to execute recursive CTE: %w", err)
	}
	return count > 0, nil
}
