// Copyright (C) 2024 Rafael Galvan

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

import "strconv"

// Player represents a high level definition of a server Player.
type Client struct {
	// PlayerID is the primary key of the Player
	PlayerID int
	// Role specifies the player role from the server
	Role int
	// Name is the player name
	Name string
	// PlayerState specifies the player state
	PlayerState int
	// ServerID is the foreign key
	ServerID int
	// Server is who the Player belongs to
	Server Server `gorm:"foreignKey:ServerID;constraint:OnDelete:CASCADE"`
}

func (r *Client) ID() string {
	return strconv.Itoa(r.PlayerID)
}
