package repository

import (
	"fmt"
	"github.com/pressly/goose/v3"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Config содержит параметры для подключения к PostgreSQL
type Config struct {
	Host          string
	Port          string
	Username      string
	Password      string
	DBName        string
	SSLMode       string
	MigrationPath string
}

type Store struct {
	DB *gorm.DB
	DepartmentRepository
	EmployeeRepository
	log *zap.Logger
}

// NewStore создает новое подключение к БД через GORM
func NewStore(cfg Config, log *zap.Logger) (*Store, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.Username, cfg.Password, cfg.DBName, cfg.SSLMode)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Error("Failed to connect to database", zap.Error(err))
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Error("Failed to get database instance", zap.Error(err))
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}

	log.Info("Testing connection to database")
	if err := sqlDB.Ping(); err != nil {
		log.Error("Failed to ping database", zap.Error(err))
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}
	log.Info("Database connection established successfully")

	log.Info("Running migrations", zap.String("path", cfg.MigrationPath))
	if err := goose.SetDialect("postgres"); err != nil {
		log.Error("Failed to set goose dialect", zap.Error(err))
		return nil, fmt.Errorf("failed to set goose dialect: %w", err)
	}
	if err := goose.Up(sqlDB, cfg.MigrationPath); err != nil {
		log.Error("Failed to run migrations", zap.Error(err))
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}
	log.Info("Migrations applied successfully")

	return &Store{
		DB:                   db,
		DepartmentRepository: *NewDepartmentRepository(db, log),
		EmployeeRepository:   *NewEmployeeRepository(db, log),
		log:                  log.Named("Repository"),
	}, nil
}

func (r *Store) Close() error {
	sqlDB, err := r.DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance for close: %w", err)
	}
	return sqlDB.Close()
}
