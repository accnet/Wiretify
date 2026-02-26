package database

import (
	"log"
	"wiretify/internal/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDB(dbPath string) error {
	var err error
	DB, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		return err
	}

	log.Println("Migrating database...")
	err = DB.AutoMigrate(&models.Peer{}, &models.Setting{}, &models.PortForward{}, &models.Domain{}, &models.Endpoint{})
	if err != nil {
		return err
	}

	return nil
}
