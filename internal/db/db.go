package db

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
	"github.com/glebarez/sqlite"

	"gorm.io/gorm"
)

func SetupDB() (*gorm.DB, error) {
	dir := filepath.Join(xdg.StateHome, "meshok")

	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}

	dbFile := filepath.Join(dir, "meshok.db")
	dsn := fmt.Sprintf("file:%s?_pragma=_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&foreign_keys(1)", dbFile)

	g, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// TODO: run migrations here

	return g, nil
}
