package domain

import "testing"

func TestRoleScopes_AdminHasWildcard(t *testing.T) {
	if !RoleAdmin.Scopes().Has("*") {
		t.Error("admin should have wildcard scope")
	}
}

func TestRoleScopes_UserHasServerAndApiKey(t *testing.T) {
	s := RoleUser.Scopes()
	for _, a := range []Action{
		ActionServerList, ActionServerRead, ActionServerWrite,
		ActionServerDelete, ActionServerGrant,
		ActionApiKeyList, ActionApiKeyRead, ActionApiKeyWrite, ActionApiKeyDelete,
	} {
		if !s.HasAction(a) {
			t.Errorf("user role missing %q", a)
		}
	}
	if s.Has("*") {
		t.Error("user role should not have wildcard")
	}
}

func TestRoleScopes_GuestIsReadOnly(t *testing.T) {
	s := RoleGuest.Scopes()
	if !s.HasAction(ActionServerList) || !s.HasAction(ActionServerRead) {
		t.Error("guest should be able to list and read servers")
	}
	if s.HasAction(ActionServerWrite) || s.HasAction(ActionApiKeyRead) {
		t.Error("guest should not have write or api_key scopes")
	}
}

func TestRoleScopes_UnknownRoleEmpty(t *testing.T) {
	if !Role("nonsense").Scopes().Empty() {
		t.Error("unknown role should yield empty scope set")
	}
}

func TestRoleOf(t *testing.T) {
	staff := User{IsStaff: true}
	if RoleOf(staff) != RoleAdmin {
		t.Error("staff user should map to admin role")
	}
	regular := User{IsStaff: false}
	if RoleOf(regular) != RoleUser {
		t.Error("non-staff user should map to user role")
	}
}
