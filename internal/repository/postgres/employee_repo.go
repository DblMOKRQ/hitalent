package repository

import (
	"context"
	"fmt"
	"go.uber.org/zap"

	"gorm.io/gorm"
	"hitalent/internal/domain"
)

type EmployeeRepository struct {
	db  *gorm.DB
	log *zap.Logger
}

func NewEmployeeRepository(db *gorm.DB, log *zap.Logger) *EmployeeRepository {
	return &EmployeeRepository{
		db:  db,
		log: log.Named("EmployeeRepository"),
	}
}

// Create создает нового сотрудника
func (r *EmployeeRepository) Create(ctx context.Context, emp *domain.Employee) error {
	r.log.Debug("Create Employee", zap.String("Full Name", emp.FullName))
	if err := r.db.WithContext(ctx).Create(emp).Error; err != nil {
		return fmt.Errorf("failed to create employee: %w", err)
	}
	r.log.Debug("Create Employee success", zap.String("Full Name", emp.FullName))
	return nil
}
