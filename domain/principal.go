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

// CredentialKind labels which authentication path produced a Principal.
// Useful for audit logs and for policy decisions that should distinguish
// browser sessions from machine credentials.
type CredentialKind string

const (
	// CredentialLogin is a browser/dashboard session. Today this is the
	// ApiLoginKey; once XF SSO is wired up it will be an XF access token.
	CredentialLogin CredentialKind = "login"
	// CredentialApi is a Carbon-issued API key (game client, game server,
	// or personal access key).
	CredentialApi CredentialKind = "api"
)

// Principal is the unified identity of the caller for a single request.
// It is built once by the auth middleware and consumed by the policy layer
// and handlers. Handlers should never reach back into the credential row
// behind the principal — every authorization fact they need lives here.
type Principal struct {
	// UserID is the XenForo user id that owns or backs this principal.
	UserID int

	// CredentialKind identifies how the principal was authenticated.
	CredentialKind CredentialKind

	// CredentialID is the database id of the credential row, used for
	// audit logging ("which key did this"). Zero for principals not
	// backed by a row (e.g. unauthenticated guests, if introduced).
	CredentialID int

	// User is the full XF user record fetched at auth time. Stored on the
	// principal so handlers (e.g. /users/me) don't need a second round-trip.
	User User

	// Role is the user-level role, derived from the User record at auth
	// time (see RoleOf).
	Role Role

	// Scopes is the effective scope set: the credential's declared scopes
	// intersected with the user's role scopes. Never expanded later.
	Scopes ScopeSet

	// IP is the source address the request was received from.
	IP string

	// BoundKind is the kind of entity this credential is restricted to,
	// e.g. "server". Empty means unbound — the principal can act on any
	// entity its scopes and ownership permit. Set from ApiKey.TargetKind.
	BoundKind string

	// BoundID is the entity id the credential is bound to. Must accompany
	// BoundKind; together they restrict the principal to a single target.
	BoundID string
}

// IsBound reports whether the principal is restricted to a single entity.
// Bound principals fail collection-level checks and any single-entity
// check whose target doesn't match the binding.
func (p Principal) IsBound() bool {
	return p.BoundKind != "" && p.BoundID != ""
}

// IsAdmin reports whether the principal has the universal wildcard scope.
// Equivalent to "is staff" today; phrased as a scope check so audits and
// policy decisions don't special-case admin.
func (p Principal) IsAdmin() bool {
	return p.Scopes.Has("*")
}
