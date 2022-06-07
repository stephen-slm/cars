package repository

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Client struct {
	DB *gorm.DB
}

func NewRepository(connectionUrl string) (Repository, error) {
	db, err := gorm.Open(postgres.Open(connectionUrl), &gorm.Config{})

	if err != nil {
		return nil, err
	}

	_ = db.AutoMigrate(&Execution{})

	return Client{DB: db}, nil
}

type Repository interface {
	InsertExecution(execution *Execution) error
	UpdateExecution(id string, columns Execution) (bool, error)
}
