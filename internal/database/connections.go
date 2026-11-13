package database

import (
	"context"
	"errors"
	"slices"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type DeviceDb struct {
	db *gorm.DB
}

func NewDeviceStore(db *gorm.DB) DeviceDb {
	return DeviceDb{
		db: db,
	}
}

type Device struct {
	// random generated string
	Id string `gorm:"primaryKey"`
	// instance name
	InstanceName string `gorm:"column:instance_name"`
	// user defined pseudoname, if empty use InstanceName instead
	PseudoName string `gorm:"column:pseudo_name"`
	// list of device's ip addresses
	IPs datatypes.JSONSlice[string] `gorm:"column:ips;type:text;not null;default:'[]'"`

	//TODO: add auth-related keys (secrets? trust flags?)
}

func (Device) TableName() string {
	return "found_devices"
}

func (db *DeviceDb) UpsertDevice(ctx context.Context, d Device) error {
	err := db.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing Device
		err := tx.First(&existing, "id = ?", d.Id).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return tx.Create(&d).Error
		}
		if err != nil {
			return err
		}

		// If a remove client was restarted with a new instance name,
		// update it locally
		if d.InstanceName != "" && d.InstanceName != existing.InstanceName {
			if err := tx.Model(&existing).Update("instance_name", d.InstanceName).Error; err != nil {
				return err
			}
		}

		// add fresh IP addresses
		merged := slices.Concat(existing.IPs, d.IPs)
		slices.Sort(merged)
		merged = slices.Compact(merged)

		return tx.Model(&existing).Update("ips", datatypes.JSONSlice[string](merged)).Error
	})

	if err != nil {
		return err
	}

	return nil
}

func (db *DeviceDb) GetDeviceById(ctx context.Context, id string) (*Device, error) {
	var d Device
	err := db.db.WithContext(ctx).First(&d, "id = ?", id).Error

	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (db *DeviceDb) RemoveDeviceAddresses(ctx context.Context, id string, addr ...string) error {
	if addr == nil {
		return nil
	}

	err := db.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var d Device
		if err := tx.First(&d, "id = ?", id).Error; err != nil {
			return err
		}
		ips := slices.DeleteFunc(slices.Clone(d.IPs), func(s string) bool {
			return slices.Contains(addr, s)
		})
		return tx.Model(&d).Update("ips", datatypes.JSONSlice[string](ips)).Error
	})

	if err != nil {
		return err
	}

	return nil
}
