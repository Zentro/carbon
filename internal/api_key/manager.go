package api_key

import (
	"carbon/domain"
	"context"
	"crypto/rand"
	"encoding/hex"

	"gorm.io/gorm"
)

type Manager struct {
	db *gorm.DB
}

func NewManager(ctx context.Context, db *gorm.DB) (*Manager, error) {
	m := &Manager{db: db}
	return m, nil
}

func (m *Manager) FindByKey(key string) (*domain.ApiKey, error) {
	var apiKey domain.ApiKey
	if err := m.db.First(&apiKey, "`key` = ?", key).Error; err != nil {
		return nil, err
	}
	return &apiKey, nil
}

func (m *Manager) FindByUser(user int) ([]*domain.ApiKey, error) {
	var apiKeys []*domain.ApiKey
	if err := m.db.Where("user_id = ?", user).Find(&apiKeys).Error; err != nil {
		return nil, err
	}
	return apiKeys, nil
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

// GenerateRandomKey will generate a cryptographically random 64 char key which
// should always be unique.
func GenerateRandomKey() (string, error) {
	bytes := make([]byte, 32) // 32 bytes = 64 hex characters
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
