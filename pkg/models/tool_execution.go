package models

import (
	"time"

	"gorm.io/gorm"
)

type ToolExecution struct {
	ID           uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	CreatedAt    time.Time      `json:"created_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
	SessionID    string         `gorm:"type:varchar(64);index" json:"session_id,omitempty"`
	ToolName     string         `gorm:"type:varchar(255);index;not null" json:"tool_name"`
	InputJSON    string         `gorm:"type:text" json:"input_json"`
	OutputJSON   string         `gorm:"type:text" json:"output_json,omitempty"`
	ErrorMessage string         `gorm:"type:text" json:"error_message,omitempty"`
	DurationMs   int64          `json:"duration_ms"`
	Success      bool           `gorm:"index" json:"success"`
}
