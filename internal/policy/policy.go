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

// Package policy holds Carbon's authorization layer. It evaluates whether
// a Principal may perform an Action against an optional target Manageable.
// The rules are deliberately small and live in one function: scope, then
// admin wildcard, then ownership. Per-kind grant storage will plug in here
// in a follow-up slice.
package policy

import (
	"context"
	"errors"

	"carbon/domain"
)

var (
	// ErrInsufficientScope means the principal lacks the action in its
	// effective scope set. Surfaced as 403 by middleware.
	ErrInsufficientScope = errors.New("policy: insufficient scope")

	// ErrForbidden means the principal has the scope but is not the owner
	// (and grants don't yet exist to delegate ownership). Surfaced as 403.
	ErrForbidden = errors.New("policy: forbidden")

	// ErrKindMismatch means the action's kind prefix doesn't match the
	// target's Kind(). Indicates a route-wiring bug, not an auth failure.
	ErrKindMismatch = errors.New("policy: action/target kind mismatch")
)

// Authorizer evaluates Can(principal, action, target). The zero value is
// usable; state is added as grant stores arrive in subsequent slices.
type Authorizer struct{}

// New returns a ready-to-use Authorizer.
func New() *Authorizer {
	return &Authorizer{}
}

// Can reports whether the principal may perform action against target. A
// nil target means a collection-level action (e.g. list, create); only the
// scope check applies there. Non-nil targets get an additional ownership
// check (and, eventually, a grant lookup).
//
// Bound credentials (api keys with TargetKind/TargetID set) are evaluated
// strictly: they may only act on the bound (kind, id), and only while the
// credential's owner still owns that entity. A transferred or re-owned
// entity automatically invalidates every bound key that references it.
// Bound principals deliberately cannot perform collection-level actions —
// listing or creating is the unbound owner's job.
func (a *Authorizer) Can(_ context.Context, p domain.Principal, action domain.Action, target domain.Manageable) error {
	// 1. Scope gate. Cheap, often-fail, no DB.
	if !p.Scopes.HasAction(action) {
		return ErrInsufficientScope
	}

	// 2. Bound principals can never perform collection actions.
	if target == nil {
		if p.IsBound() {
			return ErrForbidden
		}
		return nil
	}

	// 3. Action and target must agree on kind. Catches route-wiring bugs.
	if action.Kind() != target.Kind() {
		return ErrKindMismatch
	}

	// 4. Bound credential: target must match the binding, and the
	//    credential's owner must still own the target. No admin override —
	//    bindings are deliberately stricter than ownership and a bound key
	//    is intended for one specific entity.
	if p.IsBound() {
		if target.Kind() != p.BoundKind || target.EntityID() != p.BoundID {
			return ErrForbidden
		}
		if target.OwnerID() != p.UserID {
			return ErrForbidden
		}
		return nil
	}

	// 5. Admin wildcard wins for unbound principals.
	if p.IsAdmin() {
		return nil
	}

	// 6. Ownership.
	if target.OwnerID() != 0 && target.OwnerID() == p.UserID {
		return nil
	}

	// TODO(grants): consult per-kind grant store before returning forbidden,
	// so co-admins / delegated access can pass. Tracked as a follow-up slice.

	return ErrForbidden
}
