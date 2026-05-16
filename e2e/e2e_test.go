package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	repository "hitalent/internal/repository/postgres"
	"hitalent/internal/service"
	"hitalent/internal/transport/http/handlers"
	"hitalent/internal/transport/http/router"
)

func setupTestServer(t *testing.T) (*httptest.Server, *repository.Store) {
	log := zap.NewNop() // Отключаем логи для чистых логов в тестах

	dbConf := repository.Config{
		Host:          "localhost",
		Port:          "5432",
		Username:      "postgres",
		Password:      "123",
		DBName:        "hitalent",
		SSLMode:       "disable",
		MigrationPath: "../migrations", // Путь к миграциям относительно папки tests
	}

	store, err := repository.NewStore(dbConf, log)
	require.NoError(t, err)

	err = store.DB.Exec("TRUNCATE TABLE employees, departments RESTART IDENTITY CASCADE;").Error
	require.NoError(t, err)

	deptService := service.NewDepartmentService(&store.DepartmentRepository, log)
	empService := service.NewEmployeeService(&store.EmployeeRepository, &store.DepartmentRepository, log)

	deptHandler := handlers.NewDepartmentHandler(deptService)
	empHandler := handlers.NewEmployeeHandler(empService)

	mux := router.SetupRoutes(deptHandler, empHandler, log)

	server := httptest.NewServer(mux)
	return server, store
}

func makeRequest(t *testing.T, server *httptest.Server, method, path string, body interface{}) (*http.Response, map[string]interface{}) {
	var reqBody []byte
	var err error
	if body != nil {
		reqBody, err = json.Marshal(body)
		require.NoError(t, err)
	}

	req, err := http.NewRequest(method, server.URL+path, bytes.NewBuffer(reqBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)

	var respBody map[string]interface{}
	if resp.StatusCode != http.StatusNoContent {
		err = json.NewDecoder(resp.Body).Decode(&respBody)
		require.NoError(t, err)
	}
	resp.Body.Close()

	return resp, respBody
}

func TestE2E_API(t *testing.T) {
	server, store := setupTestServer(t)
	defer server.Close()
	defer store.Close()

	var rootDeptID int
	var childDeptID int

	t.Run("1. Create Department (Root)", func(t *testing.T) {
		body := map[string]interface{}{
			"name": "  IT Department  ", // Проверка на тримминг
		}
		resp, respBody := makeRequest(t, server, "POST", "/departments/", body)
		defer resp.Body.Close()
		require.Equal(t, http.StatusCreated, resp.StatusCode)
		require.Equal(t, "IT Department", respBody["name"])
		rootDeptID = int(respBody["id"].(float64))
	})

	t.Run("2. Create Child Department", func(t *testing.T) {
		body := map[string]interface{}{
			"name":      "Backend",
			"parent_id": rootDeptID,
		}
		resp, respBody := makeRequest(t, server, "POST", "/departments/", body)
		defer resp.Body.Close()
		require.Equal(t, http.StatusCreated, resp.StatusCode)
		childDeptID = int(respBody["id"].(float64))
	})

	t.Run("3. Prevent duplicate name in same parent", func(t *testing.T) {
		body := map[string]interface{}{
			"name":      "Backend",
			"parent_id": rootDeptID,
		}
		resp, _ := makeRequest(t, server, "POST", "/departments/", body)
		defer resp.Body.Close()
		require.Equal(t, http.StatusConflict, resp.StatusCode) // 409 Conflict
	})

	t.Run("4. Create Employee in Child", func(t *testing.T) {
		body := map[string]interface{}{
			"full_name": "Ivan Ivanov",
			"position":  "Go Developer",
		}
		path := fmt.Sprintf("/departments/%d/employees/", childDeptID)
		resp, respBody := makeRequest(t, server, "POST", path, body)
		defer resp.Body.Close()
		require.Equal(t, http.StatusCreated, resp.StatusCode)
		require.Equal(t, "Ivan Ivanov", respBody["full_name"])
	})

	t.Run("5. Prevent Create Employee in non-existent department", func(t *testing.T) {
		body := map[string]interface{}{
			"full_name": "John Doe",
			"position":  "Ghost",
		}
		resp, _ := makeRequest(t, server, "POST", "/departments/9999/employees/", body)
		defer resp.Body.Close()
		require.Equal(t, http.StatusNotFound, resp.StatusCode) // 404
	})

	t.Run("6. Get Department Tree", func(t *testing.T) {
		path := fmt.Sprintf("/departments/%d?depth=2&include_employees=true", rootDeptID)
		resp, respBody := makeRequest(t, server, "GET", path, nil)
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)
		require.Equal(t, "IT Department", respBody["name"])

		children := respBody["children"].([]interface{})
		require.Len(t, children, 1)

		child := children[0].(map[string]interface{})
		require.Equal(t, "Backend", child["name"])

		employees := child["employees"].([]interface{})
		require.Len(t, employees, 1)
		require.Equal(t, "Ivan Ivanov", employees[0].(map[string]interface{})["full_name"])
	})

	t.Run("7. Prevent Department as its own parent", func(t *testing.T) {
		body := map[string]interface{}{
			"parent_id": rootDeptID,
		}
		path := fmt.Sprintf("/departments/%d", rootDeptID)
		resp, _ := makeRequest(t, server, "PATCH", path, body)
		defer resp.Body.Close()
		require.Equal(t, http.StatusConflict, resp.StatusCode) // 409
	})

	t.Run("8. Prevent Cyclic Tree", func(t *testing.T) {
		// Пытаемся переместить корень (rootDeptID) внутрь его дочернего (childDeptID)
		body := map[string]interface{}{
			"parent_id": childDeptID,
		}
		path := fmt.Sprintf("/departments/%d", rootDeptID)
		resp, _ := makeRequest(t, server, "PATCH", path, body)
		defer resp.Body.Close()
		require.Equal(t, http.StatusConflict, resp.StatusCode) // 409
	})

	t.Run("9. Delete Department (reassign mode)", func(t *testing.T) {
		// Создадим новый отдел для перевода
		bodyDept := map[string]interface{}{"name": "Frontend"}
		respD, respBodyD := makeRequest(t, server, "POST", "/departments/", bodyDept)
		defer respD.Body.Close()
		require.Equal(t, http.StatusCreated, respD.StatusCode)
		frontendID := int(respBodyD["id"].(float64))

		// Удаляем Backend, переводим сотрудников в Frontend
		path := fmt.Sprintf("/departments/%d?mode=reassign&reassign_to_department_id=%d", childDeptID, frontendID)
		respDel, _ := makeRequest(t, server, "DELETE", path, nil)
		defer respDel.Body.Close()
		require.Equal(t, http.StatusNoContent, respDel.StatusCode) // 204

		// Проверяем, что сотрудник теперь во Frontend
		pathGet := fmt.Sprintf("/departments/%d", frontendID)
		respGet, respBodyGet := makeRequest(t, server, "GET", pathGet, nil)
		defer respGet.Body.Close()
		employees := respBodyGet["employees"].([]interface{})
		require.Len(t, employees, 1)
		require.Equal(t, "Ivan Ivanov", employees[0].(map[string]interface{})["full_name"])
	})

	t.Run("10. Delete Department (cascade mode)", func(t *testing.T) {
		// Удаляем корень каскадно. Должны удалиться все зависимые сущности.
		path := fmt.Sprintf("/departments/%d?mode=cascade", rootDeptID)
		respDel, _ := makeRequest(t, server, "DELETE", path, nil)
		defer respDel.Body.Close()
		require.Equal(t, http.StatusNoContent, respDel.StatusCode)
	})
}
