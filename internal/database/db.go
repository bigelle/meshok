package database

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
	"github.com/glebarez/sqlite"

	"gorm.io/gorm"
)

func SetupDB(path ...string) (*gorm.DB, error) {
	var dir string
	if path == nil {
		dir = filepath.Join(xdg.StateHome, "meshok")
	} else {
		// TODO: validate it
		dir = path[0]
	}

	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}

	dbFile := filepath.Join(dir, "meshok.db")
	dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&foreign_keys(1)", dbFile)

	g, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// TODO: run migrations here
	g.AutoMigrate(&Device{})

	return g, nil
}
