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

// ApiKey represents a high level definition of an API key.
type ApiKey struct {
	// ID is the primary key of the API key
	ID uint `gorm:"primaryKey" json:"api_key_id"`
	// USerID references the user who the API key belongs to
	UserID uint `gorm:"not null" json:"api_key_user_id"`
	// Key is the actual API key string
	Key string `gorm:"not null" json:"key"`
	// Role specifies the role associated with the API key
	Role ApiKeyRole `gorm:"not null" json:"role"`
}

const (
	// _Operator represents an API key role for operators with elevated privileges
	_Operator = "operator"
	// _User represents an API key role for regular users with limited privileges
	_User = "user"
	//_Guest represents an API key role for guests with no privileges
	_Guest = "guest"
)

// ApiKeyRole defines the role associated with an API key.
type ApiKeyRole string

// IsValid checks whether the ApiKeyRole is valid.
// A valid role is one of: operator, user, or guest.
func (apiRole ApiKeyRole) IsValid() bool {
	return apiRole == _Operator || apiRole == _Guest || apiRole == _User
}

// IsOperator checks whether the ApiKeyRole is an operator role.
func (apiRole ApiKeyRole) IsOperator() bool {
	return apiRole == _Operator
}

// IsGuest checks whether the ApiKeyRole is a guest role.
func (apiRole ApiKeyRole) IsGuest() bool {
	return apiRole == _Guest
}

// IsUser checks whether the ApiKeyRole is a user role.
func (apiRole ApiKeyRole) IsUser() bool {
	return apiRole == _User
}
