package repository

import (
	"time"
)

type Execution struct {
	ID string `gorm:"primarykey"`

	Status     string
	TestStatus string

	CompileMs int64
	RuntimeMs int64

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
