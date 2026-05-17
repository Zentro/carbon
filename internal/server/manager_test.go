package server

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"carbon/domain"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock, *sql.DB) {
	t.Helper()
	rawDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	gdb, err := gorm.Open(mysql.New(mysql.Config{
		Conn:                      rawDB,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		rawDB.Close()
		t.Fatalf("gorm.Open: %v", err)
	}
	return gdb, mock, rawDB
}

func TestNewManager(t *testing.T) {
	gdb, _, raw := newMockDB(t)
	defer raw.Close()
	m, err := NewManager(context.Background(), gdb)
	if err != nil || m == nil || m.db != gdb {
		t.Fatalf("NewManager: m=%v err=%v", m, err)
	}
}

func TestFindByID_InvalidUUID(t *testing.T) {
	gdb, _, raw := newMockDB(t)
	defer raw.Close()
	m, _ := NewManager(context.Background(), gdb)

	if _, err := m.FindByID("not-a-uuid"); err == nil {
		t.Error("expected error for invalid UUID")
	}
}

func TestFindByID(t *testing.T) {
	gdb, mock, raw := newMockDB(t)
	defer raw.Close()
	m, _ := NewManager(context.Background(), gdb)

	id := uuid.New()
	rows := sqlmock.NewRows([]string{"server_id", "name"}).AddRow(id, "myserver")
	mock.ExpectQuery("SELECT .* FROM `servers` WHERE server_id = ?").
		WithArgs(id, 1).
		WillReturnRows(rows)

	got, err := m.FindByID(id.String())
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got.Name != "myserver" {
		t.Errorf("Name = %q", got.Name)
	}
}

func TestFindByID_NotFound(t *testing.T) {
	gdb, mock, raw := newMockDB(t)
	defer raw.Close()
	m, _ := NewManager(context.Background(), gdb)

	mock.ExpectQuery("SELECT .* FROM `servers`").
		WillReturnError(gorm.ErrRecordNotFound)

	_, err := m.FindByID(uuid.New().String())
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("expected ErrRecordNotFound, got %v", err)
	}
}

func TestFindByHostAndPort(t *testing.T) {
	gdb, mock, raw := newMockDB(t)
	defer raw.Close()
	m, _ := NewManager(context.Background(), gdb)

	id := uuid.New()
	rows := sqlmock.NewRows([]string{"server_id", "host", "port"}).
		AddRow(id, "h", 1234)
	mock.ExpectQuery("SELECT .* FROM `servers` WHERE host = \\? AND port = \\?").
		WithArgs("h", 1234, 1).
		WillReturnRows(rows)

	got, err := m.FindByHostAndPort("h", 1234)
	if err != nil {
		t.Fatalf("FindByHostAndPort: %v", err)
	}
	if got.Host != "h" || got.Port != 1234 {
		t.Errorf("got %+v", got)
	}
}

func TestCreate(t *testing.T) {
	gdb, mock, raw := newMockDB(t)
	defer raw.Close()
	m, _ := NewManager(context.Background(), gdb)

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO `servers`").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	s := &domain.Server{
		Name: "n", Host: "h", Port: 1, Version: "v",
		Terrain: "t", Description: "d", MaxClients: 1, OwnerUserID: 7,
	}
	if err := m.Create(s); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if s.ServerID == uuid.Nil {
		t.Error("BeforeCreate hook should assign UUID")
	}
}

func TestCollection(t *testing.T) {
	gdb, mock, raw := newMockDB(t)
	defer raw.Close()
	m, _ := NewManager(context.Background(), gdb)

	id := uuid.New()
	srvRows := sqlmock.NewRows([]string{"server_id", "name"}).AddRow(id, "n")
	mock.ExpectQuery("SELECT .* FROM `servers`").WillReturnRows(srvRows)
	// Preload("Clients") triggers a second query.
	clientRows := sqlmock.NewRows([]string{"client_id", "server_id"})
	mock.ExpectQuery("SELECT .* FROM `clients`").
		WillReturnRows(clientRows)

	got, err := m.Collection()
	if err != nil {
		t.Fatalf("Collection: %v", err)
	}
	if len(got) != 1 {
		t.Errorf("expected 1 server, got %d", len(got))
	}
}

func TestDelete(t *testing.T) {
	// Server has no gorm.DeletedAt column, so GORM emits a hard DELETE.
	// (The doc comment on Delete claims "soft delete" — that's stale.)
	gdb, mock, raw := newMockDB(t)
	defer raw.Close()
	m, _ := NewManager(context.Background(), gdb)

	mock.ExpectBegin()
	mock.ExpectExec("DELETE FROM `servers`").
		WithArgs(1).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	if err := m.Delete(1); err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

func TestUpdate(t *testing.T) {
	gdb, mock, raw := newMockDB(t)
	defer raw.Close()
	m, _ := NewManager(context.Background(), gdb)

	id := uuid.New()
	rows := sqlmock.NewRows([]string{"server_id", "name"}).AddRow(id, "old")
	mock.ExpectQuery("SELECT .* FROM `servers` WHERE server_id = ?").
		WithArgs(id.String(), 1).
		WillReturnRows(rows)

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE `servers` SET").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	s := &domain.Server{ServerID: id, Name: "new"}
	if err := m.Update(s); err != nil {
		t.Fatalf("Update: %v", err)
	}
}

func TestUpdate_FindError(t *testing.T) {
	gdb, mock, raw := newMockDB(t)
	defer raw.Close()
	m, _ := NewManager(context.Background(), gdb)

	id := uuid.New()
	mock.ExpectQuery("SELECT .* FROM `servers`").
		WillReturnError(gorm.ErrRecordNotFound)

	err := m.Update(&domain.Server{ServerID: id, Name: "x"})
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("expected ErrRecordNotFound, got %v", err)
	}
}

func TestUpdateLastSync_InvalidUUID(t *testing.T) {
	gdb, _, raw := newMockDB(t)
	defer raw.Close()
	m, _ := NewManager(context.Background(), gdb)

	if err := m.UpdateLastSync("not-a-uuid"); err == nil {
		t.Error("expected error for invalid UUID")
	}
}

func TestUpdateLastSync(t *testing.T) {
	gdb, mock, raw := newMockDB(t)
	defer raw.Close()
	m, _ := NewManager(context.Background(), gdb)

	id := uuid.New()
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE `servers` SET `last_sync_date`").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	if err := m.UpdateLastSync(id.String()); err != nil {
		t.Fatalf("UpdateLastSync: %v", err)
	}
}
