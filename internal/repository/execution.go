package repository

import (
	"time"

	"compile-and-run-sandbox/internal/sandbox"
)

type Execution struct {
	ID string `gorm:"primarykey"`

	Source     string
	Output     string
	Status     sandbox.ContainerStatus
	TestStatus sandbox.ContainerTestStatus

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (c Client) InsertExecution(execution *Execution) error {
	result := c.DB.Create(execution)
	return result.Error
}

func (c Client) UpdateExecution(id string, columns Execution) (bool, error) {
	result := c.DB.Model(&Execution{ID: id}).Where("").Updates(columns)
	return result.RowsAffected > 0, result.Error
}
