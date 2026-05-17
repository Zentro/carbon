package user

import (
	"context"
	"testing"

	"carbon/domain"
	"carbon/remote"
)

type stubClient struct{ remote.Client }

func TestNewManager_StoresClient(t *testing.T) {
	c := &stubClient{}
	m, err := NewManager(context.Background(), c)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	if m == nil {
		t.Fatal("NewManager returned nil")
	}
	if m.client != c {
		t.Error("NewManager did not store the provided client")
	}
}

// Sanity: domain.User exists and has the expected fields. This guards against
// accidental schema renames that would break the (currently unused) user
// manager wiring.
func TestUserShape(t *testing.T) {
	u := domain.User{UserID: 1, Name: "x"}
	if u.UserID != 1 || u.Name != "x" {
		t.Error("domain.User fields not assignable")
	}
}
