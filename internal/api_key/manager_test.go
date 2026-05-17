package api_key

import (
	"context"
	"database/sql"
	"encoding/hex"
	"errors"
	"testing"
	"time"

	"carbon/domain"

	"github.com/DATA-DOG/go-sqlmock"
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

func TestFindByKey(t *testing.T) {
	gdb, mock, raw := newMockDB(t)
	defer raw.Close()
	m, _ := NewManager(context.Background(), gdb)

	rows := sqlmock.NewRows([]string{"api_key_id", "user_id", "key", "enabled"}).
		AddRow(1, 7, "abcd", true)
	mock.ExpectQuery("SELECT .* FROM `api_keys`").
		WithArgs("abcd", 1).
		WillReturnRows(rows)

	got, err := m.FindByKey("abcd")
	if err != nil {
		t.Fatalf("FindByKey: %v", err)
	}
	if got.ApiKeyID != 1 || got.UserID != 7 || got.Key != "abcd" {
		t.Errorf("unexpected key: %+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestFindByKey_NotFound(t *testing.T) {
	gdb, mock, raw := newMockDB(t)
	defer raw.Close()
	m, _ := NewManager(context.Background(), gdb)

	mock.ExpectQuery("SELECT .* FROM `api_keys`").
		WithArgs("missing", 1).
		WillReturnError(gorm.ErrRecordNotFound)

	_, err := m.FindByKey("missing")
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("expected ErrRecordNotFound, got %v", err)
	}
}

func TestFindByID(t *testing.T) {
	gdb, mock, raw := newMockDB(t)
	defer raw.Close()
	m, _ := NewManager(context.Background(), gdb)

	rows := sqlmock.NewRows([]string{"api_key_id", "user_id"}).AddRow(42, 7)
	mock.ExpectQuery("SELECT .* FROM `api_keys` WHERE api_key_id = ?").
		WithArgs(42, 1).
		WillReturnRows(rows)

	got, err := m.FindByID(42)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got.ApiKeyID != 42 {
		t.Errorf("ApiKeyID = %d, want 42", got.ApiKeyID)
	}
}

func TestFindByUser(t *testing.T) {
	gdb, mock, raw := newMockDB(t)
	defer raw.Close()
	m, _ := NewManager(context.Background(), gdb)

	rows := sqlmock.NewRows([]string{"api_key_id", "user_id"}).
		AddRow(1, 7).AddRow(2, 7)
	mock.ExpectQuery("SELECT .* FROM `api_keys` WHERE user_id = ?").
		WithArgs(7).
		WillReturnRows(rows)

	got, err := m.FindByUser(7)
	if err != nil {
		t.Fatalf("FindByUser: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("expected 2 keys, got %d", len(got))
	}
}

func TestCollection(t *testing.T) {
	gdb, mock, raw := newMockDB(t)
	defer raw.Close()
	m, _ := NewManager(context.Background(), gdb)

	rows := sqlmock.NewRows([]string{"api_key_id", "user_id"}).AddRow(1, 7).AddRow(2, 8)
	mock.ExpectQuery("SELECT .* FROM `api_keys`").WillReturnRows(rows)

	got, err := m.Collection()
	if err != nil {
		t.Fatalf("Collection: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("expected 2 keys, got %d", len(got))
	}
}

func TestCreate(t *testing.T) {
	gdb, mock, raw := newMockDB(t)
	defer raw.Close()
	m, _ := NewManager(context.Background(), gdb)

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO `api_keys`").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	k := &domain.ApiKey{UserID: 7, Key: "abc", Enabled: true, LastUsedAt: time.Now()}
	if err := m.Create(k); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestDelete(t *testing.T) {
	gdb, mock, raw := newMockDB(t)
	defer raw.Close()
	m, _ := NewManager(context.Background(), gdb)

	mock.ExpectBegin()
	mock.ExpectExec("DELETE FROM `api_keys` WHERE `key` = ?").
		WithArgs("abc").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	if err := m.Delete("abc"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

func TestDeleteByID(t *testing.T) {
	gdb, mock, raw := newMockDB(t)
	defer raw.Close()
	m, _ := NewManager(context.Background(), gdb)

	mock.ExpectBegin()
	mock.ExpectExec("DELETE FROM `api_keys` WHERE api_key_id = ?").
		WithArgs(99).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	if err := m.DeleteByID(99); err != nil {
		t.Fatalf("DeleteByID: %v", err)
	}
}

func TestGenerateRandomKey(t *testing.T) {
	k1, err := GenerateRandomKey()
	if err != nil {
		t.Fatalf("GenerateRandomKey: %v", err)
	}
	if len(k1) != 64 {
		t.Errorf("len = %d, want 64", len(k1))
	}
	if _, err := hex.DecodeString(k1); err != nil {
		t.Errorf("output is not valid hex: %v", err)
	}

	k2, _ := GenerateRandomKey()
	if k1 == k2 {
		t.Error("two calls returned identical keys — RNG broken?")
	}
}
