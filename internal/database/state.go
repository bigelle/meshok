package database

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

type StateValue struct {
	Key   string `gorm:"primaryKey,<-:create"`
	Value string `gorm:"<-"`
}

func (StateValue) TableName() string {
	return "app_state"
}

type StateStore struct {
	db *gorm.DB
}

func NewSettingsStore(g *gorm.DB) *StateStore {
	return &StateStore{
		db: g,
	}
}

func (s *StateStore) GetAllStateValues(ctx context.Context) ([]StateValue, error) {
	var settings []StateValue
	if err := s.db.Find(&settings).Error; err != nil {
		return nil, err
	}
	return settings, nil
}

func (s *StateStore) StoreStateValue(ctx context.Context, sv StateValue) error {
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var existing StateValue
		err := tx.First(&existing, "key = ?", sv.Key).Error

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return tx.Create(&sv).Error
		}
		if err != nil {
			return err
		}

		return tx.Model(&existing).Update("value", sv.Value).Error
	})
	return err
}
