package domain

import (
	"testing"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func TestServer_ManageableMethods(t *testing.T) {
	id := uuid.New()
	s := &Server{ServerID: id, OwnerUserID: 99}
	if s.Kind() != "server" {
		t.Errorf("Kind = %q, want %q", s.Kind(), "server")
	}
	if s.OwnerID() != 99 {
		t.Errorf("OwnerID = %d, want 99", s.OwnerID())
	}
	if s.EntityID() != id.String() {
		t.Errorf("EntityID = %q, want %q", s.EntityID(), id.String())
	}
	if s.ID() != id.String() {
		t.Errorf("ID = %q, want %q", s.ID(), id.String())
	}
}

func TestServer_BeforeCreate_AssignsUUIDWhenNil(t *testing.T) {
	s := &Server{}
	if err := s.BeforeCreate(&gorm.DB{}); err != nil {
		t.Fatalf("BeforeCreate: %v", err)
	}
	if s.ServerID == uuid.Nil {
		t.Error("BeforeCreate should assign a UUID when ServerID is Nil")
	}
}

func TestServer_BeforeCreate_PreservesExistingUUID(t *testing.T) {
	existing := uuid.New()
	s := &Server{ServerID: existing}
	if err := s.BeforeCreate(&gorm.DB{}); err != nil {
		t.Fatalf("BeforeCreate: %v", err)
	}
	if s.ServerID != existing {
		t.Errorf("BeforeCreate overwrote existing UUID: got %s, want %s", s.ServerID, existing)
	}
}

func TestServer_PowerStatus(t *testing.T) {
	s := &Server{ServerState: StatusOffline}
	if s.GetPowerStatus() != StatusOffline {
		t.Errorf("GetPowerStatus = %q, want offline", s.GetPowerStatus())
	}
	s.SetPowerStatus(StatusOnline)
	if s.GetPowerStatus() != StatusOnline {
		t.Errorf("after SetPowerStatus, GetPowerStatus = %q, want online", s.GetPowerStatus())
	}
}

func TestServerStatus_Predicates(t *testing.T) {
	cases := []struct {
		st               ServerStatus
		valid, online    bool
		crashed, hidden  bool
	}{
		{StatusOnline, true, true, false, false},
		{StatusOffline, true, false, false, false},
		{StatusHidden, true, true, false, true},
		{StatusCrashed, false, false, true, false},
		{ServerStatus("bogus"), false, false, false, false},
	}
	for _, tc := range cases {
		t.Run(string(tc.st), func(t *testing.T) {
			if got := tc.st.IsValid(); got != tc.valid {
				t.Errorf("IsValid = %v, want %v", got, tc.valid)
			}
			if got := tc.st.IsOnline(); got != tc.online {
				t.Errorf("IsOnline = %v, want %v", got, tc.online)
			}
			if got := tc.st.IsCrashed(); got != tc.crashed {
				t.Errorf("IsCrashed = %v, want %v", got, tc.crashed)
			}
			if got := tc.st.IsHidden(); got != tc.hidden {
				t.Errorf("IsHidden = %v, want %v", got, tc.hidden)
			}
		})
	}
}

func TestClient_ID(t *testing.T) {
	c := &Client{ClientID: 1234}
	if c.ID() != "1234" {
		t.Errorf("Client.ID = %q, want %q", c.ID(), "1234")
	}
}
