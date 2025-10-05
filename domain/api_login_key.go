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
	"carbon/config"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ApiLoginKey represens a high level definition of an API login key.
// The API login key is seperate distinct from the API key. It has no roles or scopes,
// and is purely used for the sake of users logging in and retrieving their profile.
// It should never be used to make changes to a user's profile as the lack of scope
// restrictions could let a user arbitrarily change their roles using the API's XenForo
// super key.
type ApiLoginKey struct {
	// ApiLoginkeyID is the primary key of the API login key.
	ApiLoginKeyID uint `gorm:"primaryKey" json:"api_login_key_id"`
	// UserID references the user who the API login key belongs to.
	UserID int `gorm:"not null" json:"user_id"`
	// LoginKey is the actual key string
	LoginKey string `gorm:"size:255;not null;unique" json:"api_login_key"`
	// LoginkeyExpiresAt is the time at which the API login key is no longer valid.
	LoginKeyExpiresAt time.Time `gorm:"not null" json:"api_login_key_expires_at"`
	// RefreshKey is the key string for refreshing the key itself.
	RefreshKey string `gorm:"size:255;not null;unique" json:"api_refresh_key"`
	// RefreshKeyExpresAt is the time at which the refresh key is no longer valid, and the login
	// key itself can no longer be refreshed.
	RefreshKeyExpiresAt time.Time `gorm:"not null" json:"api_refresh_key_expires_at"`
	// IP is the source of the request, if this does not match then this key should no longer be
	// considered valid.
	IP string `gorm:"not null" json:"ip_address"`
	// CreatedAt specifies the time of creation for this key.
	// Automatically managed by GORM.
	CreatedAt time.Time
	// UpdatedAt specifies the time of update for this key.
	// Automatically managed by GORM.
	UpdatedAt time.Time
	// DeletedAt specifies the time of soft deletion for this key.
	// Automatically managed by GORM.
	DeletedAt time.Time
}

// ApiLoginClaims represents the claims used in the JWT for the API login key.
type ApiLoginClaims struct {
	// UserID is the ID of the user associated with the API login key.
	UserID int `json:"user_id"`
	// Standard JWT claims.
	jwt.RegisteredClaims
}

// NewJwtKey creates a new JWT key with the given claims.
func NewJwtKey(userID int, expiresAt time.Time) (string, error) {
	claims := ApiLoginClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	key := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return key.SignedString([]byte(config.Get().Secret))
}

// NewApiLoginKey creates a new API login key.
func NewApiLoginKey(userID int, ip string) (*ApiLoginKey, error) {
	// Generate a random 32 byte key for the API login key that expires in 24 hours.
	loginKey, err := NewJwtKey(userID, time.Now().Add(24*time.Hour))
	if err != nil {
		return nil, err
	}

	// Generate a random 32 byte key for the API refresh key that expires in 7 days.
	refreshKey, err := NewJwtKey(userID, time.Now().Add(24*7*time.Hour))
	if err != nil {
		return nil, err
	}

	return &ApiLoginKey{
		UserID:              userID,
		LoginKey:            loginKey,
		LoginKeyExpiresAt:   time.Now().Add(24 * time.Hour),
		RefreshKey:          refreshKey,
		RefreshKeyExpiresAt: time.Now().Add(24 * 7 * time.Hour),
		IP:                  ip,
	}, nil
}
