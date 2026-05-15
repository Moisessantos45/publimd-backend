package db

import (
	"fmt"
	"log"
	"os"
	"publimd/internal/shared/models"
)

func InitializeDatabase() error {
	if DB == nil {
		return fmt.Errorf("database not initialized")
	}

	prod := os.Getenv("GO_ENV")
	if prod == "production" {
		fmt.Println("Running in production mode, skipping database migration")
		return nil
	}

	log.Println("Running in development mode, performing database migration")
	if err := DB.AutoMigrate(models.Models...); err != nil {
		return fmt.Errorf("error migrating database: %v", err)
	}

	if err := SeedDatabase(); err != nil {
		return fmt.Errorf("error seeding database: %v", err)
	}

	return nil
}
