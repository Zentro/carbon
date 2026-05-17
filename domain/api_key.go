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

import (
	"strconv"
	"time"
)

// ApiKey represents a high level definition of an API key.
type ApiKey struct {
	// ApiKeyID is the primary key of the API key
	ApiKeyID int `gorm:"primaryKey" json:"api_key_id,omitempty"`
	// UserID references the user who the API key belongs to
	UserID int `gorm:"not null;index" json:"api_key_user_id" binding:"required"`
	// Name is a human-readable label for the key (e.g. "My laptop", "CI bot",
	// "Server: foo"). Shown in the dashboard's key list. Optional.
	Name string `gorm:"size:255" json:"name,omitempty"`
	// Key is the actual API key string
	Key string `gorm:"not null;uniqueIndex" json:"api_key_key,omitempty"`
	// Scopes is the explicit capability set of this credential. At auth time
	// the principal's effective scopes become user.Role.Scopes() ∩ Scopes,
	// so a key can never grant authority its owner has lost. An empty
	// Scopes is treated as "every scope the user's role grants" — i.e. an
	// unbound personal key inherits the owner's full authority.
	Scopes ScopeSet `gorm:"type:text" json:"scopes,omitempty"`
	// TargetKind binds this key to a specific entity kind ("server", …).
	// When set together with TargetID, the policy layer rejects any
	// request that targets a different entity. Nil for unbound keys.
	TargetKind *string `gorm:"size:32;index:idx_api_key_target,priority:1" json:"target_kind,omitempty"`
	// TargetID is the bound entity's primary key as a string (UUID for
	// servers). Must be set together with TargetKind.
	TargetID *string `gorm:"size:64;index:idx_api_key_target,priority:2" json:"target_id,omitempty"`
	// Enabled specifies whether the key is enabled or disabled.
	Enabled bool `gorm:"not null" json:"enabled"`
	// LastUsedAt specifies the last time this key was used.
	LastUsedAt time.Time `json:"last_used_at"`
	// CreatedAt specifies the time of creation for this key.
	// Automatically managed by GORM.
	CreatedAt time.Time
	// UpdatedAt specifies the time of update for this key.
	// Automatically managed by GORM.
	UpdatedAt time.Time
}

// Kind implements Manageable.
func (k *ApiKey) Kind() string { return "api_key" }

// OwnerID implements Manageable. The owner of an API key is the user it
// was issued to.
func (k *ApiKey) OwnerID() int { return k.UserID }

// EntityID implements Manageable.
func (k *ApiKey) EntityID() string { return strconv.Itoa(k.ApiKeyID) }

// IsBound reports whether the key is restricted to a specific entity. A
// bound key authorizes only against its TargetKind/TargetID; an unbound
// key acts on whatever the owner's role permits.
func (k *ApiKey) IsBound() bool {
	return k.TargetKind != nil && k.TargetID != nil &&
		*k.TargetKind != "" && *k.TargetID != ""
}
