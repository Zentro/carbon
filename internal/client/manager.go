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

func (m *Manager) FindByIDForServer(s *domain.Server, id int) (*domain.Client, error) {
	var c domain.Client
	if err := m.db.Model(&s).Preload("Clients", "id = ?", id).First(&c).Error; err != nil {
		return nil, err
	}

	return &c, nil
}

func (m *Manager) CollectionForServer(sid int) (*domain.Client, error) {
	var c domain.Client
	if err := m.db.Where("server_id = ?", sid).Find(&c).Error; err != nil {
		return nil, err
	}

	return &c, nil
}

func (m *Manager) FindByID(id int) (*domain.Client, error) {
	var c domain.Client
	if err := m.db.First(&c, id).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

func (m *Manager) Collection() ([]*domain.Client, error) {
	var c []*domain.Client
	if err := m.db.Find(&c).Error; err != nil {
		return nil, err
	}
	return c, nil
}

func (m *Manager) Delete(id int) error {
	if err := m.db.Delete(&domain.Client{}, id).Error; err != nil {
		return err
	}
	return nil
}
