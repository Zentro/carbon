package policy

import (
	"context"
	"errors"
	"testing"

	"carbon/domain"

	"github.com/google/uuid"
)

func newAdmin() domain.Principal {
	return domain.Principal{
		UserID: 1,
		Scopes: domain.NewScopeSet("*"),
	}
}

func newUser(id int) domain.Principal {
	return domain.Principal{
		UserID: id,
		Scopes: domain.RoleUser.Scopes(),
	}
}

func newBoundKeyPrincipal(userID int, kind, id string) domain.Principal {
	p := newUser(userID)
	p.BoundKind = kind
	p.BoundID = id
	return p
}

func ownedServer(ownerID int) *domain.Server {
	return &domain.Server{
		ServerID:    uuid.New(),
		OwnerUserID: ownerID,
	}
}

func TestCan_InsufficientScope(t *testing.T) {
	a := New()
	p := domain.Principal{Scopes: domain.NewScopeSet()}
	err := a.Can(context.Background(), p, domain.ActionServerRead, nil)
	if !errors.Is(err, ErrInsufficientScope) {
		t.Errorf("expected ErrInsufficientScope, got %v", err)
	}
}

func TestCan_CollectionAction_UnboundAllowed(t *testing.T) {
	a := New()
	err := a.Can(context.Background(), newUser(5), domain.ActionServerList, nil)
	if err != nil {
		t.Errorf("collection action for unbound user should pass: %v", err)
	}
}

func TestCan_CollectionAction_BoundForbidden(t *testing.T) {
	a := New()
	p := newBoundKeyPrincipal(5, "server", "abc")
	err := a.Can(context.Background(), p, domain.ActionServerList, nil)
	if !errors.Is(err, ErrForbidden) {
		t.Errorf("bound principal must not run collection actions, got %v", err)
	}
}

func TestCan_KindMismatch(t *testing.T) {
	a := New()
	// server action, api_key target — wiring bug
	target := &domain.ApiKey{UserID: 5}
	err := a.Can(context.Background(), newUser(5), domain.ActionServerRead, target)
	if !errors.Is(err, ErrKindMismatch) {
		t.Errorf("expected ErrKindMismatch, got %v", err)
	}
}

func TestCan_Admin_BypassesOwnership(t *testing.T) {
	a := New()
	srv := ownedServer(999) // someone else's server
	if err := a.Can(context.Background(), newAdmin(), domain.ActionServerWrite, srv); err != nil {
		t.Errorf("admin should bypass ownership, got %v", err)
	}
}

func TestCan_Owner_Allowed(t *testing.T) {
	a := New()
	srv := ownedServer(7)
	if err := a.Can(context.Background(), newUser(7), domain.ActionServerWrite, srv); err != nil {
		t.Errorf("owner should be allowed, got %v", err)
	}
}

func TestCan_NonOwner_Forbidden(t *testing.T) {
	a := New()
	srv := ownedServer(7)
	err := a.Can(context.Background(), newUser(99), domain.ActionServerWrite, srv)
	if !errors.Is(err, ErrForbidden) {
		t.Errorf("non-owner should be forbidden, got %v", err)
	}
}

func TestCan_ZeroOwner_Forbidden(t *testing.T) {
	a := New()
	srv := ownedServer(0)
	err := a.Can(context.Background(), newUser(7), domain.ActionServerWrite, srv)
	if !errors.Is(err, ErrForbidden) {
		t.Error("zero owner should fail ownership check")
	}
}

func TestCan_Bound_MatchingTargetAndOwner(t *testing.T) {
	a := New()
	srv := ownedServer(7)
	p := newBoundKeyPrincipal(7, "server", srv.EntityID())
	if err := a.Can(context.Background(), p, domain.ActionServerWrite, srv); err != nil {
		t.Errorf("bound principal on matching target should pass: %v", err)
	}
}

func TestCan_Bound_WrongTarget_Forbidden(t *testing.T) {
	a := New()
	srv := ownedServer(7)
	p := newBoundKeyPrincipal(7, "server", uuid.New().String())
	err := a.Can(context.Background(), p, domain.ActionServerWrite, srv)
	if !errors.Is(err, ErrForbidden) {
		t.Errorf("bound principal on wrong target should be forbidden, got %v", err)
	}
}

func TestCan_Bound_WrongKind_Forbidden(t *testing.T) {
	a := New()
	srv := ownedServer(7)
	// Bound to a different kind entirely.
	p := newBoundKeyPrincipal(7, "race", srv.EntityID())
	err := a.Can(context.Background(), p, domain.ActionServerWrite, srv)
	if !errors.Is(err, ErrForbidden) {
		t.Errorf("bound principal with mismatched kind should be forbidden, got %v", err)
	}
}

func TestCan_Bound_OwnerTransferred_Forbidden(t *testing.T) {
	a := New()
	srv := ownedServer(99) // server is owned by someone else now
	p := newBoundKeyPrincipal(7, "server", srv.EntityID())
	err := a.Can(context.Background(), p, domain.ActionServerWrite, srv)
	if !errors.Is(err, ErrForbidden) {
		t.Errorf("bound credential should fail when owner no longer matches, got %v", err)
	}
}

func TestCan_Bound_AdminBindingStillStrict(t *testing.T) {
	// A bound credential should not get admin override. The principal here
	// has the wildcard scope (e.g. an admin user issued themselves a bound
	// key) but is still restricted to the bound target.
	a := New()
	p := newAdmin()
	p.BoundKind = "server"
	p.BoundID = uuid.New().String()

	// Different target than the binding.
	srv := ownedServer(p.UserID)
	err := a.Can(context.Background(), p, domain.ActionServerWrite, srv)
	if !errors.Is(err, ErrForbidden) {
		t.Errorf("bound admin should still be restricted to binding, got %v", err)
	}
}
