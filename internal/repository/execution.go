package repository

import (
	"time"
)

type Execution struct {
	ID string `gorm:"primarykey"`

	Language   string
	Status     string
	TestStatus string

	CompileMs       int64
	RuntimeMs       int64
	RuntimeMemoryMb int64

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (c Client) InsertExecution(execution *Execution) error {
	result := c.DB.Create(execution)
	return result.Error
}

func (c Client) UpdateExecution(id string, columns *Execution) (bool, error) {
	result := c.DB.Model(&Execution{ID: id}).Where("").Updates(columns)
	return result.RowsAffected > 0, result.Error
}

func (c Client) GetExecution(id string) (Execution, error) {
	execution := Execution{}

	result := c.DB.Where("id = ?", id).First(&execution)
	return execution, result.Error
}

func (c Client) UpdateExecutionStatus(id string, status string) error {
	result := c.DB.Where("id = ?", id).UpdateColumns(Execution{Status: status})
	return result.Error
}
