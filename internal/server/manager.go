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
	log.Info("initializing server schema...")

	if err := m.db.AutoMigrate(&domain.Server{}); err != nil {
		return err
	}

	return nil
}

func (m *Manager) Find(s *domain.Server) {

}

func (m *Manager) Create(s *domain.Server) {

}

func (m *Manager) Collection() []*domain.Server {
	return nil
}
