package service

import (
"context"
"errors"
"testing"

"github.com/stretchr/testify/require"
"go.uber.org/zap"

"hitalent/internal/domain"
)

type departmentRepoMock struct {
createFn             func(ctx context.Context, dept *domain.Department) error
getByIDFn            func(ctx context.Context, id int) (*domain.Department, error)
getWithTreeFn        func(ctx context.Context, id int, depth int, includeEmployees bool) (*domain.Department, error)
updateFn             func(ctx context.Context, dept *domain.Department) error
deleteFn             func(ctx context.Context, id int) error
reassignEmployeesFn  func(ctx context.Context, fromDeptID int, toDeptID int) error
checkDuplicateNameFn func(ctx context.Context, name string, parentID *int) (bool, error)
isDescendantFn       func(ctx context.Context, targetDeptID int, potentialParentID int) (bool, error)
}

func (m *departmentRepoMock) Create(ctx context.Context, dept *domain.Department) error {
if m.createFn != nil {
return m.createFn(ctx, dept)
}
return nil
}

func (m *departmentRepoMock) GetByID(ctx context.Context, id int) (*domain.Department, error) {
if m.getByIDFn != nil {
return m.getByIDFn(ctx, id)
}
return &domain.Department{ID: id, Name: "default"}, nil
}

func (m *departmentRepoMock) GetWithTree(ctx context.Context, id int, depth int, includeEmployees bool) (*domain.Department, error) {
if m.getWithTreeFn != nil {
return m.getWithTreeFn(ctx, id, depth, includeEmployees)
}
return &domain.Department{ID: id, Name: "default"}, nil
}

func (m *departmentRepoMock) Update(ctx context.Context, dept *domain.Department) error {
if m.updateFn != nil {
return m.updateFn(ctx, dept)
}
return nil
}

func (m *departmentRepoMock) Delete(ctx context.Context, id int) error {
if m.deleteFn != nil {
return m.deleteFn(ctx, id)
}
return nil
}

func (m *departmentRepoMock) ReassignEmployees(ctx context.Context, fromDeptID int, toDeptID int) error {
if m.reassignEmployeesFn != nil {
return m.reassignEmployeesFn(ctx, fromDeptID, toDeptID)
}
return nil
}

func (m *departmentRepoMock) CheckDuplicateName(ctx context.Context, name string, parentID *int) (bool, error) {
if m.checkDuplicateNameFn != nil {
return m.checkDuplicateNameFn(ctx, name, parentID)
}
return false, nil
}

func (m *departmentRepoMock) IsDescendant(ctx context.Context, targetDeptID int, potentialParentID int) (bool, error) {
if m.isDescendantFn != nil {
return m.isDescendantFn(ctx, targetDeptID, potentialParentID)
}
return false, nil
}

func TestDepartmentServiceCreate_TrimAndValidateName(t *testing.T) {
repo := &departmentRepoMock{}
svc := NewDepartmentService(repo, zap.NewNop())

dept := &domain.Department{Name: "   "}
err := svc.Create(context.Background(), dept)

require.ErrorIs(t, err, domain.ErrInvalidInput)
}

func TestDepartmentServiceCreate_ReturnsConflictForDuplicateName(t *testing.T) {
repo := &departmentRepoMock{
checkDuplicateNameFn: func(ctx context.Context, name string, parentID *int) (bool, error) {
require.Equal(t, "Backend", name)
return true, nil
},
}
svc := NewDepartmentService(repo, zap.NewNop())

dept := &domain.Department{Name: "  Backend  "}
err := svc.Create(context.Background(), dept)

require.ErrorIs(t, err, domain.ErrConflict)
}

func TestDepartmentServiceGetByIDWithTree_ClampsDepthToMax(t *testing.T) {
calledDepth := 0
repo := &departmentRepoMock{
getWithTreeFn: func(ctx context.Context, id int, depth int, includeEmployees bool) (*domain.Department, error) {
calledDepth = depth
return &domain.Department{ID: id, Name: "IT"}, nil
},
}
svc := NewDepartmentService(repo, zap.NewNop())

_, err := svc.GetByIDWithTree(context.Background(), 1, 99, true)

require.NoError(t, err)
require.Equal(t, 5, calledDepth)
}

func TestDepartmentServiceUpdate_ReturnsCyclicTreeError(t *testing.T) {
newParentID := 2
repo := &departmentRepoMock{
getByIDFn: func(ctx context.Context, id int) (*domain.Department, error) {
if id == 1 {
return &domain.Department{ID: 1, Name: "Root"}, nil
}
if id == 2 {
return &domain.Department{ID: 2, Name: "Child", ParentID: intPtr(1)}, nil
}
return nil, domain.ErrNotFound
},
isDescendantFn: func(ctx context.Context, targetDeptID int, potentialParentID int) (bool, error) {
return true, nil
},
}
svc := NewDepartmentService(repo, zap.NewNop())

_, err := svc.Update(context.Background(), 1, nil, &newParentID)

require.ErrorIs(t, err, domain.ErrCyclicTree)
}

func TestDepartmentServiceDelete_ReassignModeRequiresTargetID(t *testing.T) {
repo := &departmentRepoMock{
getByIDFn: func(ctx context.Context, id int) (*domain.Department, error) {
return &domain.Department{ID: id, Name: "Dept"}, nil
},
}
svc := NewDepartmentService(repo, zap.NewNop())

err := svc.Delete(context.Background(), 1, "reassign", nil)

require.ErrorIs(t, err, domain.ErrInvalidInput)
}

func TestDepartmentServiceCreate_ReturnsNotFoundWhenParentMissing(t *testing.T) {
parentID := 42
repo := &departmentRepoMock{
getByIDFn: func(ctx context.Context, id int) (*domain.Department, error) {
return nil, domain.ErrNotFound
},
}
svc := NewDepartmentService(repo, zap.NewNop())

err := svc.Create(context.Background(), &domain.Department{Name: "Dept", ParentID: &parentID})

require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestDepartmentServiceUpdate_ReturnsWrappedErrorOnGetByIDFailure(t *testing.T) {
repo := &departmentRepoMock{
getByIDFn: func(ctx context.Context, id int) (*domain.Department, error) {
return nil, errors.New("db down")
},
}
svc := NewDepartmentService(repo, zap.NewNop())

_, err := svc.Update(context.Background(), 1, nil, nil)

require.Error(t, err)
require.Contains(t, err.Error(), "failed to get department by id")
}

func intPtr(v int) *int {
return &v
}
