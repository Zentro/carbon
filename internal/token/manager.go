// Copyright (C) 2022-2024 Rafael Galvan

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package token

import (
	"carbon/domain"
	"context"
	"time"

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
	log.Info("initializing token schema into the database...")

	if err := m.db.AutoMigrate(&domain.Token{}); err != nil {
		return err
	}

	return nil
}

func (m *Manager) AsyncPurgeDb(ctx context.Context) error {
	log.Info("purging invalid tokens from the database...")

	now := time.Now()

	// Find all tokens where both tokens are expired
	var tokens []domain.Token
	if err := m.db.WithContext(ctx).Where("login_token_expires_at < ? AND refresh_token_expires_at < ?", now, now).Find(&tokens).Error; err != nil {
		return err
	}

	// Soft delete the expired tokens
	for _, token := range tokens {
		if err := m.db.WithContext(ctx).Delete(&token).Error; err != nil {
			return err
		}
	}

	return nil
}

func (m *Manager) FindByID(id uint) (domain.Token, error) {
	var token domain.Token
	if err := m.db.First(&token, id).Error; err != nil {
		return domain.Token{}, err
	}
	return token, nil
}

func (m *Manager) FindByToken(tokenValue string) (domain.Token, error) {
	var token domain.Token
	if err := m.db.Where("login_token = ?", tokenValue).First(&token).Error; err != nil {
		return domain.Token{}, err
	}
	return token, nil
}

func (m *Manager) Create(s *domain.Token) error {
	if err := m.db.Create(&s).Error; err != nil {
		return err
	}
	return nil
}

func (m *Manager) Collection() ([]*domain.Token, error) {
	var tokens []*domain.Token
	if err := m.db.Find(&tokens).Error; err != nil {
		return nil, err
	}
	return tokens, nil
}

// Invalidate sets the ExpiresAt field to now to mark the token as invalid
func (m *Manager) Invalidate(id uint) error {
	var token domain.Token
	if err := m.db.First(&token, id).Error; err != nil {
		return err
	}

	// Set the expiration time to now. This will allow us to purge it later.
	token.LoginTokenExpiresAt = time.Now()
	token.RefreshTokenExpiresAt = time.Now()
	if err := m.db.Save(&token).Error; err != nil {
		return err
	}
	return nil
}

func (m *Manager) Refresh(t *domain.Token) error {
	if err := m.db.Save(&t).Error; err != nil {
		return err
	}
	return nil
}
