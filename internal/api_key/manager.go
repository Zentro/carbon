package api_key

import (
	"carbon/domain"
	"context"

	"github.com/apex/log"
	"gorm.io/gorm"
)

type Manager struct {
	db *gorm.DB
}

func NewManager(ctx context.Context, db *gorm.DB) (*Manager, error) {
	m := &Manager{db: db}
	err := m.init()
	return m, err
}

func (m *Manager) init() error {
	log.Info("initializing api key schema into the database...")

	if err := m.db.AutoMigrate(&domain.ApiKey{}); err != nil {
		return err
	}

	return nil
}

func (m *Manager) FindByKey(key string) (*domain.ApiKey, error) {
	var apiKey domain.ApiKey
	if err := m.db.First(&apiKey, "`key` = ?", key).Error; err != nil {
		return nil, err
	}
	return &apiKey, nil
}

func (m *Manager) Create(apiKey *domain.ApiKey) error {
	if err := m.db.Create(&apiKey).Error; err != nil {
		return err
	}
	return nil
}

func (m *Manager) Collection() ([]*domain.ApiKey, error) {
	var apiKeys []*domain.ApiKey
	if err := m.db.Find(&apiKeys).Error; err != nil {
		return nil, err
	}
	return apiKeys, nil
}

func (m *Manager) Delete(key string) error {
	if err := m.db.Delete(&domain.ApiKey{}, "`key` = ?", key).Error; err != nil {
		return err
	}
	return nil
}
