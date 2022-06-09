package repository

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Client struct {
	DB *gorm.DB
}

func NewRepository(connectionURL string) (Repository, error) {
	db, err := gorm.Open(postgres.Open(connectionURL), &gorm.Config{})

	if err != nil {
		return nil, err
	}

	// ping test
	if pingErr := pingTest(db); pingErr != nil {
		return nil, pingErr
	}

	migrateErr := db.AutoMigrate(&Execution{})

	return Client{DB: db}, migrateErr
}

type Repository interface {
	InsertExecution(execution *Execution) error
	UpdateExecution(id string, columns *Execution) (bool, error)
	UpdateExecutionStatus(id string, status string) error
	GetExecution(id string) (Execution, error)
}

func pingTest(db *gorm.DB) error {
	genDB, err := db.DB()

	if err != nil {
		return err
	}

	return genDB.Ping()
}
