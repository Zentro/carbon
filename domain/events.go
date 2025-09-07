// Copyright (C) 2025 Rafael Galvan

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

type EventTerrain struct {
	EventTerrainID string `gorm:"type:uuid;primaryKey" json:"event_terrain_id"`
	ResourceId     string `json:"resource_id" binding:"required"`
	ManagerId      string `json:"manager_id" binding:"required"`

	EventCompetition []EventCompetition `gorm:"foreignKey:EventTerrainID;constraint:OnDelete:CASCADE;"`
}

type EventCompetition struct {
	EventCompetitionID string `gorm:"type:uuid;primaryKey" json:"event_competition_id"`
	Name               string `gorm:"size:255;not null" json:"name" binding:"required"`
	Description        string `gorm:"type:text;not null" json:"description" binding:"required"`

	EventTerrainID string       `gorm:"type:uuid;index" json:"event_terrain_id"`
	EventTerrain   EventTerrain `gorm:"foreignKey:EventTerrainID"`

	Races []EventRace `gorm:"foreignKey:EventCompetitionID;constraint:OnDelete:CASCADE;"`
}

type EventRace struct {
	EventRaceID string `gorm:"type:uuid;primaryKey" json:"event_race_id"`
	UserID      int    `json:"user_id"`

	EventCompetitionID string           `gorm:"type:uuid;index" json:"competitionId"`
	EventCompetition   EventCompetition `gorm:"foreignKey:EventCompetitionID"`
}
