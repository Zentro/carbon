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

// Manageable is implemented by any entity that can be the target of an
// authorization check. The policy layer uses the three methods to decide:
//
//   - whether the action's prefix matches the entity's Kind (route guard),
//   - whether the principal owns the entity (OwnerID),
//   - and which row to consult for grants (EntityID).
//
// Adding a new manageable kind (race, event, …) is just: implement this
// interface, define its Action constants, and register a grant store.
// The Authorizer.Can method does not change.
type Manageable interface {
	// Kind is the resource-kind string ("server", "race", …). Must match
	// the prefix of any Action checked against this entity.
	Kind() string

	// OwnerID is the user id of the entity's creator/owner. Zero is
	// treated as "no owner" and will fail every ownership check.
	OwnerID() int

	// EntityID is the entity's primary key as a string, used to look up
	// grants in the per-kind grant store.
	EntityID() string
}
