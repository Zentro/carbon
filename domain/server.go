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
	"strconv"
)

type Server struct {
	ServerID    int          `gorm:"primaryKey;autoIncrement" json:"server_id,omitempty"`
	ServerState ServerStatus `gorm:"type:smallint" json:"server_state,omitempty"`
	Name        string       `gorm:"size:255;not null" json:"name" binding:"required"`
	IP          string       `gorm:"size:255;not null" json:"ip" binding:"required"`
	Port        int          `gorm:"not null" json:"port" binding:"required"`
	Version     string       `gorm:"size:100;not null" json:"version" binding:"required"`
	Description string       `gorm:"type:text;not null" json:"description" binding:"required"`
	IconUrl     string       `gorm:"size:255" json:"icon_url"`
	OwnerID     int          `gorm:"not null" json:"owner_id"`
	HasPassword *bool        `gorm:"not null" json:"has_password" binding:"required"`
	MaxClients  uint         `gorm:"not null" json:"max_clients" binding:"required"`
	IsVisible   *bool        `gorm:"not null" json:"is_visible" binding:"required"`
	ServerDate  uint         `gorm:"autoCreateTime" json:"server_date,omitempty"`
}

func (r *Server) ID() string {
	return strconv.Itoa(r.ServerID)
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

func (s *Server) SetPowerStatus(status ServerStatus) error {
	switch status {
	case StatusOnline:
		if s.ServerState == StatusOnline {
			return nil
		}
		return nil
	case StatusOffline:
		if s.ServerState == StatusOffline {
			return nil
		}
		return nil
	case StatusHidden:
		return nil
	case StatusCrashed:
		if s.ServerState != StatusCrashed {
			return nil
		}
		s.ServerState = StatusCrashed
		return nil
	}
	//s.ServerState =
	return nil
}
