package storage

import (
	"context"
	"fmt"

	"github.com/tb0hdan/wass-mcp/pkg/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type SQLiteStorage struct {
	db *gorm.DB
}

type Config struct {
	DatabasePath string
	Debug        bool
}

func NewSQLiteStorage(cfg Config) (*SQLiteStorage, error) {
	logLevel := logger.Silent
	if cfg.Debug {
		logLevel = logger.Info
	}

	database, err := gorm.Open(sqlite.Open(cfg.DatabasePath), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect database: %w", err)
	}

	// Auto-migrate schema
	if err := database.AutoMigrate(&models.ToolExecution{}); err != nil {
		return nil, fmt.Errorf("failed to migrate schema: %w", err)
	}

	return &SQLiteStorage{db: database}, nil
}

func (s *SQLiteStorage) CreateToolExecution(ctx context.Context, exec *models.ToolExecution) error {
	return s.db.WithContext(ctx).Create(exec).Error
}

func (s *SQLiteStorage) GetToolExecution(ctx context.Context, id uint) (*models.ToolExecution, error) {
	var exec models.ToolExecution
	err := s.db.WithContext(ctx).First(&exec, id).Error
	if err != nil {
		return nil, err
	}
	return &exec, nil
}

func (s *SQLiteStorage) GetToolExecutions(ctx context.Context, limit, offset int) ([]models.ToolExecution, int64, error) {
	var executions []models.ToolExecution
	var total int64

	s.db.WithContext(ctx).Model(&models.ToolExecution{}).Count(&total)

	query := s.db.WithContext(ctx).Order("created_at DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}
	err := query.Find(&executions).Error
	return executions, total, err
}

func (s *SQLiteStorage) GetToolExecutionsBySession(ctx context.Context, sessionID string) ([]models.ToolExecution, error) {
	var executions []models.ToolExecution
	err := s.db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Order("created_at DESC").
		Find(&executions).Error
	return executions, err
}

func (s *SQLiteStorage) GetToolExecutionsByTool(ctx context.Context, toolName string, limit int) ([]models.ToolExecution, error) {
	var executions []models.ToolExecution
	query := s.db.WithContext(ctx).
		Where("tool_name = ?", toolName).
		Order("created_at DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Find(&executions).Error
	return executions, err
}

func (s *SQLiteStorage) DeleteToolExecution(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Delete(&models.ToolExecution{}, id).Error
}

func (s *SQLiteStorage) DeleteAllToolExecutions(ctx context.Context) error {
	return s.db.WithContext(ctx).Where("1 = 1").Delete(&models.ToolExecution{}).Error
}

func (s *SQLiteStorage) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
