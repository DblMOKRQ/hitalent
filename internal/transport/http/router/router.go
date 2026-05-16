package router

import (
	"go.uber.org/zap"
	"hitalent/internal/transport/http/handlers"
	"hitalent/internal/transport/http/middleware"
	"net/http"
)

func SetupRoutes(deptHandler *handlers.DepartmentHandler, empHandler *handlers.EmployeeHandler, log *zap.Logger) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /departments/", deptHandler.Create)
	mux.HandleFunc("GET /departments/{id}", deptHandler.Get)
	mux.HandleFunc("PATCH /departments/{id}", deptHandler.Update)
	mux.HandleFunc("DELETE /departments/{id}", deptHandler.Delete)

	mux.HandleFunc("POST /departments/{id}/employees/", empHandler.Create)

	loggingMiddleware := middleware.LoggingMiddleware(log.Named("middleware"))
	return loggingMiddleware(mux)
}
