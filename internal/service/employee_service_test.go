package service

import (
"context"
"errors"
"testing"

"github.com/stretchr/testify/require"
"go.uber.org/zap"

"hitalent/internal/domain"
)

type employeeRepoMock struct {
createFn func(ctx context.Context, emp *domain.Employee) error
}

func (m *employeeRepoMock) Create(ctx context.Context, emp *domain.Employee) error {
if m.createFn != nil {
return m.createFn(ctx, emp)
}
return nil
}

type departmentProviderMock struct {
getByIDFn func(ctx context.Context, id int) (*domain.Department, error)
}

func (m *departmentProviderMock) GetByID(ctx context.Context, id int) (*domain.Department, error) {
if m.getByIDFn != nil {
return m.getByIDFn(ctx, id)
}
return &domain.Department{ID: id, Name: "Dept"}, nil
}

func TestEmployeeServiceCreate_InvalidFullName(t *testing.T) {
svc := NewEmployeeService(&employeeRepoMock{}, &departmentProviderMock{}, zap.NewNop())

err := svc.Create(context.Background(), &domain.Employee{DepartmentID: 1, FullName: "   ", Position: "Dev"})

require.ErrorIs(t, err, domain.ErrInvalidInput)
}

func TestEmployeeServiceCreate_InvalidPosition(t *testing.T) {
svc := NewEmployeeService(&employeeRepoMock{}, &departmentProviderMock{}, zap.NewNop())

err := svc.Create(context.Background(), &domain.Employee{DepartmentID: 1, FullName: "Ivan", Position: "   "})

require.ErrorIs(t, err, domain.ErrInvalidInput)
}

func TestEmployeeServiceCreate_ReturnsNotFoundWhenDepartmentMissing(t *testing.T) {
deptMock := &departmentProviderMock{
getByIDFn: func(ctx context.Context, id int) (*domain.Department, error) {
return nil, domain.ErrNotFound
},
}
svc := NewEmployeeService(&employeeRepoMock{}, deptMock, zap.NewNop())

err := svc.Create(context.Background(), &domain.Employee{DepartmentID: 10, FullName: "Ivan", Position: "Dev"})

require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestEmployeeServiceCreate_TrimsAndCreatesEmployee(t *testing.T) {
created := false
repo := &employeeRepoMock{
createFn: func(ctx context.Context, emp *domain.Employee) error {
created = true
require.Equal(t, "Ivan Ivanov", emp.FullName)
require.Equal(t, "Go Developer", emp.Position)
return nil
},
}
svc := NewEmployeeService(repo, &departmentProviderMock{}, zap.NewNop())

err := svc.Create(context.Background(), &domain.Employee{DepartmentID: 1, FullName: "  Ivan Ivanov ", Position: " Go Developer  "})

require.NoError(t, err)
require.True(t, created)
}

func TestEmployeeServiceCreate_WrapsUnexpectedDepartmentError(t *testing.T) {
deptMock := &departmentProviderMock{
getByIDFn: func(ctx context.Context, id int) (*domain.Department, error) {
return nil, errors.New("db unavailable")
},
}
svc := NewEmployeeService(&employeeRepoMock{}, deptMock, zap.NewNop())

err := svc.Create(context.Background(), &domain.Employee{DepartmentID: 1, FullName: "Ivan", Position: "Dev"})

require.Error(t, err)
require.Contains(t, err.Error(), "failed to check department by id")
}
