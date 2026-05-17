package domain

import "testing"

func TestPrincipal_IsBound(t *testing.T) {
	cases := []struct {
		name string
		p    Principal
		want bool
	}{
		{"both set", Principal{BoundKind: "server", BoundID: "abc"}, true},
		{"kind only", Principal{BoundKind: "server"}, false},
		{"id only", Principal{BoundID: "abc"}, false},
		{"neither", Principal{}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.p.IsBound(); got != tc.want {
				t.Errorf("IsBound = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestPrincipal_IsAdmin(t *testing.T) {
	admin := Principal{Scopes: NewScopeSet("*")}
	if !admin.IsAdmin() {
		t.Error("principal with wildcard scope should be admin")
	}

	notAdmin := Principal{Scopes: NewScopeSet("server:*")}
	if notAdmin.IsAdmin() {
		t.Error("kind wildcard should not be considered admin")
	}

	empty := Principal{}
	if empty.IsAdmin() {
		t.Error("empty principal should not be admin")
	}
}
