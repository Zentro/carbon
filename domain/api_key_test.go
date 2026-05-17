package domain

import "testing"

func strPtr(s string) *string { return &s }

func TestApiKey_ManageableMethods(t *testing.T) {
	k := &ApiKey{ApiKeyID: 42, UserID: 7}
	if k.Kind() != "api_key" {
		t.Errorf("Kind = %q, want %q", k.Kind(), "api_key")
	}
	if k.OwnerID() != 7 {
		t.Errorf("OwnerID = %d, want 7", k.OwnerID())
	}
	if k.EntityID() != "42" {
		t.Errorf("EntityID = %q, want %q", k.EntityID(), "42")
	}
}

func TestApiKey_IsBound(t *testing.T) {
	cases := []struct {
		name string
		key  ApiKey
		want bool
	}{
		{"unbound", ApiKey{}, false},
		{"both set", ApiKey{TargetKind: strPtr("server"), TargetID: strPtr("uuid")}, true},
		{"only kind", ApiKey{TargetKind: strPtr("server")}, false},
		{"only id", ApiKey{TargetID: strPtr("uuid")}, false},
		{"both empty strings", ApiKey{TargetKind: strPtr(""), TargetID: strPtr("")}, false},
		{"kind empty string", ApiKey{TargetKind: strPtr(""), TargetID: strPtr("uuid")}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.key.IsBound(); got != tc.want {
				t.Errorf("IsBound = %v, want %v", got, tc.want)
			}
		})
	}
}
