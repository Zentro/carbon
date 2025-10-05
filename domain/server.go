// Copyright (C) 2022, 2025 Rafael Galvan

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
	"strconv"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Server presents a high level definition of a game server.
// It contains all the information needed to connect to and manage the server.
type Server struct {
	// ServerID is the primary key of the server.
	ServerID uuid.UUID `gorm:"type:uuid;primaryKey;" json:"server_id,omitempty"`

	// ServerState represents the current state of the server.
	ServerState ServerStatus `gorm:"not null;default:'offline'" json:"server_state,omitempty"`

	// Name is the name of the server.
	Name string `gorm:"size:255;not null" json:"name" binding:"required"`

	// Host is the server's hostname or IP address.
	Host string `gorm:"size:255;not null;uniqueIndex:idx_host_port" json:"host" binding:"required"`

	// Port is the server's port number.
	Port int `gorm:"not null;uniqueIndex:idx_host_port" json:"port" binding:"required"`

	// Version is the version of RoRnet the server is running.
	Version string `gorm:"size:100;not null" json:"version" binding:"required"`

	// Terrain is the terrain the server is running.
	Terrain string `gorm:"not null;default:'any'" json:"terrain" binding:"required"`

	// Description is a brief description of the server.
	Description string `gorm:"type:text;not null" json:"description" binding:"required"`

	// HasPassword indicates if the server requires a password.
	HasPassword bool `gorm:"not null;default:false" json:"has_password"`

	// MaxClients is the maximum number of clients allowed on the server.
	MaxClients uint `gorm:"not null" json:"max_clients" binding:"required"`

	// Clients is the list of clients connected to the server.
	Clients []Client `gorm:"foreignKey:ServerID;constraint:OnDelete:CASCADE" json:"clients,omitempty"`

	// Visible indicates if the server is visible to users.
	Visible bool `gorm:"not null;default:true" json:"is_visible"`

	// ServerDate is the date the server was created.
	ServerDate time.Time `gorm:"autoCreateTime" json:"server_date,omitempty"`

	// LastUpdateDate is the date the server was last updated.
	LastUpdateDate time.Time `gorm:"autoUpdateTime" json:"last_update_date,omitempty"`

	// LastSyncDate is the date the server was last synchronized by the server itself.
	LastSyncDate time.Time `json:"last_sync_date,omitempty"`

	// ApiKeyID is the foreign key to the API key used to manage this server.
	// This is unique to ensure one-to-one relationship between server and API key.
	// It is also not null to ensure that a server always has an API key.
	ApiKeyID int `gorm:"uniqueIndex;not null" json:"api_key_id"`

	// ApiKey is the API key used to manage this server.
	ApiKey ApiKey `gorm:"constraint:OnDelete:CASCADE;"`
}

// BeforeCreate will run before each insert operation to make sure the UUID
// will be not NIL. This can be removed if MariaDB supports the UUID() operation.
func (s *Server) BeforeCreate(tx *gorm.DB) (err error) {
	if s.ServerID == uuid.Nil {
		s.ServerID = uuid.New()
	}
	return
}

func (s *Server) ID() string {
	return s.ServerID.String()
}

// Client represents a high level definition of a server client.
type Client struct {
	// ClientID is the primary key of the client
	ClientID int `gorm:"primaryKey;autoIncrement" json:"client_id,omitempty"`
	// Role specifies the client role from the server
	Role int
	// Name is the client name
	Name string
	// ServerID is the foreign key
	ServerID uuid.UUID `json:"server_id"`
	// Server is who the client belongs to
	Server Server `gorm:"constraint:OnDelete:CASCADE"`
	// ConnectedAt specifies the time the client connected
	ConnectedAt *time.Time
	// UpdatedAt specifies the time the client was last updated
	UpdatedAt *time.Time
}

func (r *Client) ID() string {
	return strconv.Itoa(r.ClientID)
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
