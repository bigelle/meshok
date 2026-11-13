package database

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

type Setting struct {
	Key   string `gorm:"primaryKey,<-:create"`
	Value string `gorm:"<-"`
}

type SettingsStore struct {
	db *gorm.DB
}

func NewSettingsStore(g *gorm.DB) *SettingsStore {
	return &SettingsStore{
		db: g,
	}
}

func (s *SettingsStore) LoadAllSettings(ctx context.Context) ([]Setting, error) {
	var settings []Setting
	if err := s.db.Find(&settings).Error; err != nil {
		return nil, err
	}
	return settings, nil
}

func (s *SettingsStore) StoreSetting(ctx context.Context, setting Setting) error {
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var existing Setting
		err := tx.First(&existing, "key = ?", setting.Key).Error

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return tx.Create(&setting).Error
		}
		if err != nil {
			return err
		}

		return tx.Model(&existing).Update("value", setting.Value).Error
	})
	return err
}
