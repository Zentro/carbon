// Copyright (C) 2025 Rafael Galvan <rafael.galvan@rigsofrods.org>

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

// Role is the user-level capability bundle. It lives on the user, not on
// the credential. Credentials carry their own scopes which are intersected
// with the role's scopes at authentication time.
type Role string

const (
	RoleAdmin Role = "admin"
	RoleUser  Role = "user"
	RoleGuest Role = "guest"
)

// Scopes returns the scope set granted by this role. Hardcoded in Go for now;
// move to a DB-driven mapping if and when role management needs a UI.
func (r Role) Scopes() ScopeSet {
	switch r {
	case RoleAdmin:
		return NewScopeSet("*")
	case RoleUser:
		return NewScopeSet(
			string(ActionServerList),
			string(ActionServerRead),
			string(ActionServerWrite),
			string(ActionServerDelete),
			string(ActionServerGrant),
			string(ActionApiKeyList),
			string(ActionApiKeyRead),
			string(ActionApiKeyWrite),
			string(ActionApiKeyDelete),
		)
	case RoleGuest:
		return NewScopeSet(
			string(ActionServerList),
			string(ActionServerRead),
		)
	default:
		return NewScopeSet()
	}
}

// RoleOf returns the role assigned to a user. Admins are XF staff (IsStaff);
// every other authenticated user is a regular user. Guest is reserved for
// principals built without a backing user (not currently produced).
func RoleOf(u User) Role {
	if u.IsStaff {
		return RoleAdmin
	}
	return RoleUser
}
