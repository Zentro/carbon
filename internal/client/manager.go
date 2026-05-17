// Copyright (C) 2022-2025 Rafael Galvan

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

package client

import (
	"carbon/domain"
	"context"

	"gorm.io/gorm"
)

type Manager struct {
	db *gorm.DB
}

func NewManager(ctx context.Context, db *gorm.DB) (*Manager, error) {
	m := &Manager{db: db}
	return m, nil
}

// Create adds a new client to a server in the database.
func (m *Manager) Create(c *domain.Client, s *domain.Server) error {
	if err := m.db.Model(&s).Association("Clients").Append(&c); err != nil {
		return err
	}
	return nil
}

func (m *Manager) Update(c *domain.Client) error {
	if err := m.db.Save(&c).Error; err != nil {
		return err
	}
	return nil
}

// CollectionByServerID retrieves all clients associated with a specific server ID.
func (m *Manager) CollectionByServerID(server_id string) ([]domain.Client, error) {
	var c []domain.Client
	if err := m.db.Where("server_id = ?", server_id).Find(&c).Error; err != nil {
		return nil, err
	}

	return c, nil
}

// FindByID retrieves a client from the database based on its ID.
func (m *Manager) FindByID(id int) (*domain.Client, error) {
	var c domain.Client
	if err := m.db.First(&c, id).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

// Collection retrieves all clients from the database.
func (m *Manager) Collection() ([]*domain.Client, error) {
	var c []*domain.Client
	if err := m.db.Find(&c).Error; err != nil {
		return nil, err
	}
	return c, nil
}

// Delete removes a client from the database based on its ID.
func (m *Manager) Delete(id int) error {
	if err := m.db.Delete(&domain.Client{}, id).Error; err != nil {
		return err
	}
	return nil
}
