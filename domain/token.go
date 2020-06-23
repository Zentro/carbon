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
	"carbon/config"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

type Token struct {
	ID                    uint      `gorm:"primaryKey" json:"token_id"`
	UserID                int       `gorm:"not null" json:"user_id"`
	LoginToken            string    `gorm:"size:255;not null;unique" json:"login_token"`
	LoginTokenExpiresAt   time.Time `gorm:"not null" json:"login_token_expires_at"`
	RefreshToken          string    `gorm:"size:255;not null;unique" json:"refresh_token"`
	RefreshTokenExpiresAt time.Time `gorm:"not null" json:"refresh_token_expires_at"`
	IPAddress             string    `gorm:"not null" json:"ip_address"`

	gorm.Model
}

func NewToken(uid int, ip_addr string) (*Token, error) {
	loginTokenExpiresAt := time.Now().Add(24 * time.Hour)
	refreshTokenExpiresAt := time.Now().Add(24 * 7 * time.Hour)

	loginToken, err := generateToken(uid, loginTokenExpiresAt)
	if err != nil {
		return nil, err
	}

	refreshToken, err := generateToken(uid, refreshTokenExpiresAt)
	if err != nil {
		return nil, err
	}

	return &Token{
		UserID:                uid,
		LoginToken:            loginToken,
		LoginTokenExpiresAt:   loginTokenExpiresAt,
		RefreshToken:          refreshToken,
		RefreshTokenExpiresAt: refreshTokenExpiresAt,
		IPAddress:             ip_addr,
	}, nil
}

func generateToken(userID int, expiresAt time.Time) (string, error) {
	claims := CustomClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(config.Get().Secret))
}

type CustomClaims struct {
	UserID int `json:"user_id"`
	jwt.RegisteredClaims
}
