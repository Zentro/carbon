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

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// Action is the type-level capability a principal needs to perform an
// operation against the API. Format: "<kind>:<verb>" — e.g. "server:write".
// Ownership of the specific target is checked separately by the policy layer.
type Action string

const (
	ActionServerList   Action = "server:list"
	ActionServerRead   Action = "server:read"
	ActionServerWrite  Action = "server:write"
	ActionServerDelete Action = "server:delete"
	ActionServerGrant  Action = "server:grant"

	ActionApiKeyList   Action = "api_key:list"
	ActionApiKeyRead   Action = "api_key:read"
	ActionApiKeyWrite  Action = "api_key:write"
	ActionApiKeyDelete Action = "api_key:delete"
)

// Kind returns the resource-kind prefix of the action ("server", "api_key").
// Used by the policy layer to cross-check that a target's Kind() matches the
// action's prefix — guards against route-wiring mistakes that would otherwise
// authorize an action against the wrong type of entity.
func (a Action) Kind() string {
	if i := strings.IndexByte(string(a), ':'); i >= 0 {
		return string(a)[:i]
	}
	return string(a)
}

// ScopeSet is a principal's effective set of capabilities. Membership is
// checked with Has / HasAction. The wildcard "*" matches every action;
// "<kind>:*" matches every action on a kind.
type ScopeSet map[string]struct{}

// NewScopeSet builds a ScopeSet from a list of scope strings.
func NewScopeSet(scopes ...string) ScopeSet {
	s := make(ScopeSet, len(scopes))
	for _, sc := range scopes {
		s[sc] = struct{}{}
	}
	return s
}

// Has reports whether the set contains the exact scope string.
func (s ScopeSet) Has(scope string) bool {
	_, ok := s[scope]
	return ok
}

// HasAction reports whether the set authorizes the given action, accounting
// for the "*" universal wildcard and "<kind>:*" kind wildcards.
func (s ScopeSet) HasAction(a Action) bool {
	if s.Has("*") {
		return true
	}
	if s.Has(a.Kind() + ":*") {
		return true
	}
	return s.Has(string(a))
}

// Intersect returns the scopes present in both sets. Used at authentication
// time to narrow a credential's declared scopes by the user's current role —
// so a key can never grant authority its owner has lost.
func (s ScopeSet) Intersect(other ScopeSet) ScopeSet {
	out := make(ScopeSet)
	for scope := range s {
		if scope == "*" || other.Has(scope) || other.Has("*") {
			out[scope] = struct{}{}
		}
	}
	return out
}

// Empty reports whether the set contains no scopes.
func (s ScopeSet) Empty() bool { return len(s) == 0 }

// IsSubsetOf reports whether every scope in s is authorized by other,
// honoring "*" and "<kind>:*" wildcards in `other`. Wildcards inside `s`
// are matched literally (they only count as a subset if `other` contains
// the same wildcard or a broader one).
func (s ScopeSet) IsSubsetOf(other ScopeSet) bool {
	if other.Has("*") {
		return true
	}
	for scope := range s {
		switch {
		case scope == "*":
			return false
		case other.Has(scope):
			continue
		default:
			if i := strings.IndexByte(scope, ':'); i >= 0 && other.Has(scope[:i]+":*") {
				continue
			}
			return false
		}
	}
	return true
}

// Sorted returns the scopes as a sorted slice. Stable order is useful for
// serialization and logs.
func (s ScopeSet) Sorted() []string {
	out := make([]string, 0, len(s))
	for k := range s {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// Value implements driver.Valuer so a ScopeSet can be persisted as a
// comma-separated TEXT column. Order is stable (sorted) so equal sets
// serialize identically.
func (s ScopeSet) Value() (driver.Value, error) {
	if s == nil {
		return "", nil
	}
	return strings.Join(s.Sorted(), ","), nil
}

// MarshalJSON serializes the set as a JSON array of scope strings,
// sorted for stable output. nil and empty produce an empty array so
// clients see a consistent shape.
func (s ScopeSet) MarshalJSON() ([]byte, error) {
	if s == nil {
		return []byte("[]"), nil
	}
	return json.Marshal(s.Sorted())
}

// UnmarshalJSON accepts a JSON array of scope strings. Empty array or
// null produces an empty ScopeSet (non-nil).
func (s *ScopeSet) UnmarshalJSON(b []byte) error {
	var raw []string
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	out := make(ScopeSet, len(raw))
	for _, scope := range raw {
		scope = strings.TrimSpace(scope)
		if scope != "" {
			out[scope] = struct{}{}
		}
	}
	*s = out
	return nil
}

// Scan implements sql.Scanner. Accepts string or []byte; an empty value
// produces an empty (non-nil) ScopeSet.
func (s *ScopeSet) Scan(src any) error {
	if src == nil {
		*s = make(ScopeSet)
		return nil
	}
	var raw string
	switch v := src.(type) {
	case string:
		raw = v
	case []byte:
		raw = string(v)
	default:
		return fmt.Errorf("domain: ScopeSet.Scan: unsupported source type %T", src)
	}
	out := make(ScopeSet)
	for _, p := range strings.Split(raw, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			out[p] = struct{}{}
		}
	}
	*s = out
	return nil
}
