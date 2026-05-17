package client

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

func TestFindByID(t *testing.T) {
	gdb, mock, raw := newMockDB(t)
	defer raw.Close()
	m, _ := NewManager(context.Background(), gdb)

	rows := sqlmock.NewRows([]string{"client_id", "name"}).AddRow(5, "alice")
	mock.ExpectQuery("SELECT .* FROM `clients`").
		WithArgs(5, 1).
		WillReturnRows(rows)

	got, err := m.FindByID(5)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got.ClientID != 5 || got.Name != "alice" {
		t.Errorf("unexpected client: %+v", got)
	}
}

func TestFindByID_NotFound(t *testing.T) {
	gdb, mock, raw := newMockDB(t)
	defer raw.Close()
	m, _ := NewManager(context.Background(), gdb)

	mock.ExpectQuery("SELECT .* FROM `clients`").
		WillReturnError(gorm.ErrRecordNotFound)

	_, err := m.FindByID(99)
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("expected ErrRecordNotFound, got %v", err)
	}
}

func TestCollectionByServerID(t *testing.T) {
	gdb, mock, raw := newMockDB(t)
	defer raw.Close()
	m, _ := NewManager(context.Background(), gdb)

	sid := uuid.New().String()
	rows := sqlmock.NewRows([]string{"client_id", "server_id"}).
		AddRow(1, sid).AddRow(2, sid)
	mock.ExpectQuery("SELECT .* FROM `clients` WHERE server_id = ?").
		WithArgs(sid).
		WillReturnRows(rows)

	got, err := m.CollectionByServerID(sid)
	if err != nil {
		t.Fatalf("CollectionByServerID: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("expected 2 clients, got %d", len(got))
	}
}

func TestCollection(t *testing.T) {
	gdb, mock, raw := newMockDB(t)
	defer raw.Close()
	m, _ := NewManager(context.Background(), gdb)

	rows := sqlmock.NewRows([]string{"client_id"}).AddRow(1).AddRow(2).AddRow(3)
	mock.ExpectQuery("SELECT .* FROM `clients`").WillReturnRows(rows)

	got, err := m.Collection()
	if err != nil {
		t.Fatalf("Collection: %v", err)
	}
	if len(got) != 3 {
		t.Errorf("expected 3 rows, got %d", len(got))
	}
}

func TestUpdate(t *testing.T) {
	gdb, mock, raw := newMockDB(t)
	defer raw.Close()
	m, _ := NewManager(context.Background(), gdb)

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE `clients` SET").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	c := &domain.Client{ClientID: 1, Name: "renamed"}
	if err := m.Update(c); err != nil {
		t.Fatalf("Update: %v", err)
	}
}

func TestDelete(t *testing.T) {
	gdb, mock, raw := newMockDB(t)
	defer raw.Close()
	m, _ := NewManager(context.Background(), gdb)

	mock.ExpectBegin()
	mock.ExpectExec("DELETE FROM `clients`").
		WithArgs(7).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	if err := m.Delete(7); err != nil {
		t.Fatalf("Delete: %v", err)
	}
}
