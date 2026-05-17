// Copyright (C) 2022 Rafael Galvan

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

package server

import (
	"carbon/domain"
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Manager struct {
	db *gorm.DB
}

func NewManager(ctx context.Context, db *gorm.DB) (*Manager, error) {
	m := &Manager{db: db}
	return m, nil
}

// FindByID retrieves a server from the database based on its UUID.
func (m *Manager) FindByID(id string) (*domain.Server, error) {
	uuid, err := uuid.Parse(id)
	if err != nil {
		return nil, err // Return error if the provided ID is not a valid UUID
	}

	var server domain.Server
	if err := m.db.First(&server, "server_id = ?", uuid).Error; err != nil {
		return nil, err
	}
	return &server, nil
}

// Create adds a new server to the database.
func (m *Manager) Create(server *domain.Server) error {
	if err := m.db.Create(&server).Error; err != nil {
		return err
	}
	return nil
}

// Update modifies an existing server in the database.
func (m *Manager) Update(s *domain.Server) error {
	var server domain.Server
	if err := m.db.First(&server, "server_id = ?", s.ID()).Error; err != nil {
		return err
	}

	// Update the fields that need to be changed
	if err := m.db.Model(&server).Updates(s).Error; err != nil {
		return err
	}

	return nil
}

// FindByHostAndPort retrieves a server from the database based on its host and port.
func (m *Manager) FindByHostAndPort(host string, port int) (*domain.Server, error) {
	var server domain.Server
	if err := m.db.Where("host = ? AND port = ?", host, port).First(&server).Error; err != nil {
		return nil, err
	}
	return &server, nil
}

// Collection retrieves all servers from the database.
func (m *Manager) Collection() ([]*domain.Server, error) {
	var servers []*domain.Server
	if err := m.db.Preload("Clients").Find(&servers).Error; err != nil {
		return nil, err
	}
	return servers, nil
}

// Delete removes a server from the database by ID.
// This operation is reversible as it only does a "soft delete".
func (m *Manager) Delete(id int) error {
	if err := m.db.Delete(&domain.Server{}, id).Error; err != nil {
		return err
	}
	return nil
}

// UpdateLastSync updates the last_sync_date field of a server to the current time.
func (m *Manager) UpdateLastSync(id string) error {
	uuid, err := uuid.Parse(id)
	if err != nil {
		return err // Return error if the provided ID is not a valid UUID
	}

	return m.db.Model(&domain.Server{}).Where("server_id = ?", uuid).Update("last_sync_date", sql.NullTime{
		Time:  time.Now(),
		Valid: true,
	}).Error
}
