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

package domain

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Server struct {
	ServerID       uuid.UUID    `gorm:"type:char(36);primaryKey;" json:"server_id,omitempty"` // TODO: there's a chance that MariaDB uses UUID()
	ServerState    ServerStatus `gorm:"not null;default:'offline'" json:"server_state,omitempty"`
	Name           string       `gorm:"size:255;not null" json:"name" binding:"required"`
	Host           string       `gorm:"size:255;not null;uniqueIndex:idx_host_port" json:"host" binding:"required"`
	Port           int          `gorm:"not null;uniqueIndex:idx_host_port" json:"port" binding:"required"`
	Version        string       `gorm:"size:100;not null" json:"version" binding:"required"`
	Terrain        string       `gorm:"not null;default:'any'" json:"terrain" binding:"required"` // TODO: normalize this, terrain + GUID
	Description    string       `gorm:"type:text;not null" json:"description" binding:"required"`
	IconUrl        string       `gorm:"size:255" json:"icon_url"`
	OwnerID        int          `gorm:"not null" json:"owner_id"`
	HasPassword    *bool        `gorm:"not null" json:"has_password" binding:"required"`
	MaxClients     uint         `gorm:"not null" json:"max_clients" binding:"required"`
	Clients        []Client     `gorm:"foreignKey:ServerID;constraint:OnDelete:CASCADE" json:"clients,omitempty"`
	IsVisible      *bool        `gorm:"not null" json:"is_visible" binding:"required"`
	ServerDate     uint         `gorm:"autoCreateTime" json:"server_date,omitempty"`
	LastUpdateDate uint         `gorm:"autoUpdateTime" json:"last_Update_date,omitempty"`
	LastSyncDate   uint         `json:"last_sync_date,omitempty"`
	ApiKeyID       *uint        `gorm:"unique"`
	ApiKey         ApiKey       `gorm:"constraint:OnDelete:CASCADE;"`
}

// BeforeCreate will run before each insert operation to make sure the UUID
// will be not NIL. This can be removed if MariaDB supports the UUID() operation.
func (s *Server) BeforeCreate(tx *gorm.DB) (err error) {
	if s.ServerID == uuid.Nil {
		s.ServerID = uuid.New()
	}
	return
}

func (r *Server) ID() string {
	return r.ServerID.String()
}

type ServerStatus string

const (
	StatusOnline  = "online"
	StatusOffline = "offline"
	StatusHidden  = "hidden"
	StatusCrashed = "crashed"
)

func (st ServerStatus) IsValid() bool {
	return st == StatusOnline ||
		st == StatusOffline ||
		st == StatusHidden
}

func (st ServerStatus) IsOnline() bool {
	return st == StatusOnline || st == StatusHidden
}

func (st ServerStatus) IsCrashed() bool {
	return st == StatusCrashed
}

func (st ServerStatus) IsHidden() bool {
	return st == StatusHidden
}

func (s *Server) GetPowerStatus() ServerStatus {
	return s.ServerState
}

func (s *Server) SetPowerStatus(status ServerStatus) {
	s.ServerState = status
}
