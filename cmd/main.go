package main

import (
	"context"
	"errors"
	"go.uber.org/zap"
	"hitalent/internal/config"
	repository "hitalent/internal/repository/postgres"
	"hitalent/internal/service"
	"hitalent/internal/transport/http/handlers"
	"hitalent/internal/transport/http/router"
	"hitalent/pkg/logger"
	systemLog "log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	const shutdownTimeout = 10 * time.Second
	cfg := config.MustLoad()
	log, err := logger.NewLogger(cfg.LogLevel)
	if err != nil {
		panic(err)
	}
	defer func() {
		err := log.Sync()
		if err != nil && !errors.Is(err, syscall.ENOTTY) {
			systemLog.Printf("failed to sync logger: %v", err)
		}
	}()

	dbConf := repository.Config{
		Host:          cfg.DBHost,
		Port:          cfg.DBPort,
		Username:      cfg.DBUser,
		Password:      cfg.DBPassword,
		DBName:        cfg.DBName,
		SSLMode:       cfg.SSLMode,
		MigrationPath: cfg.DBMigrationPath,
	}

	storeRepo, err := repository.NewStore(dbConf, log)
	if err != nil {
		log.Error("Failed to initialize repository", zap.Error(err))
		return
	}
	defer storeRepo.Close()
	departmentService := service.NewDepartmentService(&storeRepo.DepartmentRepository, log)
	employeeService := service.NewEmployeeService(&storeRepo.EmployeeRepository, &storeRepo.DepartmentRepository, log)

	departmentHandler := handlers.NewDepartmentHandler(departmentService)
	employeeHandler := handlers.NewEmployeeHandler(employeeService)

	mux := router.SetupRoutes(departmentHandler, employeeHandler, log)
	srv := &http.Server{
		Addr:         cfg.HTTPAddr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	serverErrors := make(chan error, 1)

	go func() {
		log.Info("Starting HTTP server", zap.String("addr", cfg.HTTPAddr))
		serverErrors <- srv.ListenAndServe()
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("Failed to start server", zap.Error(err))
		}
	case sig := <-quit:
		log.Info("Shutting down server...", zap.String("signal", sig.String()))

		ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			log.Error("Server forced to shutdown", zap.Error(err))
		} else {
			log.Info("Server gracefully stopped")
		}

		if err := storeRepo.Close(); err != nil {
			log.Error("Failed to close database connection", zap.Error(err))
		}
	}
}
