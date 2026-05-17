package api_login_key

import (
	"context"
	"database/sql"
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

func TestFindByID(t *testing.T) {
	gdb, mock, raw := newMockDB(t)
	defer raw.Close()
	m, _ := NewManager(context.Background(), gdb)

	rows := sqlmock.NewRows([]string{"api_login_key_id", "user_id", "login_key"}).
		AddRow(1, 7, "tok")
	mock.ExpectQuery("SELECT .* FROM `api_login_keys`").
		WithArgs(1, 1).
		WillReturnRows(rows)

	got, err := m.FindByID(1)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got.UserID != 7 {
		t.Errorf("UserID = %d, want 7", got.UserID)
	}
}

func TestFindByID_NotFound(t *testing.T) {
	gdb, mock, raw := newMockDB(t)
	defer raw.Close()
	m, _ := NewManager(context.Background(), gdb)

	mock.ExpectQuery("SELECT .* FROM `api_login_keys`").
		WillReturnError(gorm.ErrRecordNotFound)

	_, err := m.FindByID(99)
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("expected ErrRecordNotFound, got %v", err)
	}
}

func TestFindByToken(t *testing.T) {
	gdb, mock, raw := newMockDB(t)
	defer raw.Close()
	m, _ := NewManager(context.Background(), gdb)

	rows := sqlmock.NewRows([]string{"api_login_key_id", "user_id", "login_key"}).
		AddRow(1, 7, "abc")
	mock.ExpectQuery("SELECT .* FROM `api_login_keys` WHERE login_key = ?").
		WithArgs("abc", 1).
		WillReturnRows(rows)

	got, err := m.FindByToken("abc")
	if err != nil {
		t.Fatalf("FindByToken: %v", err)
	}
	if got.LoginKey != "abc" {
		t.Errorf("LoginKey = %q, want abc", got.LoginKey)
	}
}

func TestCreate(t *testing.T) {
	gdb, mock, raw := newMockDB(t)
	defer raw.Close()
	m, _ := NewManager(context.Background(), gdb)

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO `api_login_keys`").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	k := &domain.ApiLoginKey{
		UserID:              7,
		LoginKey:            "lk",
		RefreshKey:          "rk",
		LoginKeyExpiresAt:   time.Now().Add(time.Hour),
		RefreshKeyExpiresAt: time.Now().Add(24 * time.Hour),
		IP:                  "1.2.3.4",
	}
	if err := m.Create(k); err != nil {
		t.Fatalf("Create: %v", err)
	}
}

func TestCollection(t *testing.T) {
	gdb, mock, raw := newMockDB(t)
	defer raw.Close()
	m, _ := NewManager(context.Background(), gdb)

	rows := sqlmock.NewRows([]string{"api_login_key_id"}).AddRow(1).AddRow(2)
	mock.ExpectQuery("SELECT .* FROM `api_login_keys`").WillReturnRows(rows)

	got, err := m.Collection()
	if err != nil {
		t.Fatalf("Collection: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("expected 2 rows, got %d", len(got))
	}
}

func TestInvalidate(t *testing.T) {
	gdb, mock, raw := newMockDB(t)
	defer raw.Close()
	m, _ := NewManager(context.Background(), gdb)

	rows := sqlmock.NewRows([]string{"api_login_key_id", "user_id", "login_key", "refresh_key"}).
		AddRow(1, 7, "lk", "rk")
	mock.ExpectQuery("SELECT .* FROM `api_login_keys`").
		WithArgs(1, 1).
		WillReturnRows(rows)

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE `api_login_keys` SET").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	if err := m.Invalidate(1); err != nil {
		t.Fatalf("Invalidate: %v", err)
	}
}

func TestInvalidate_FindError(t *testing.T) {
	gdb, mock, raw := newMockDB(t)
	defer raw.Close()
	m, _ := NewManager(context.Background(), gdb)

	mock.ExpectQuery("SELECT .* FROM `api_login_keys`").
		WillReturnError(gorm.ErrRecordNotFound)

	err := m.Invalidate(1)
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("expected ErrRecordNotFound, got %v", err)
	}
}

func TestRefresh(t *testing.T) {
	gdb, mock, raw := newMockDB(t)
	defer raw.Close()
	m, _ := NewManager(context.Background(), gdb)

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE `api_login_keys` SET").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	k := &domain.ApiLoginKey{
		ApiLoginKeyID:       1,
		UserID:              7,
		LoginKey:            "lk",
		RefreshKey:          "rk",
		LoginKeyExpiresAt:   time.Now(),
		RefreshKeyExpiresAt: time.Now(),
	}
	if err := m.Refresh(k); err != nil {
		t.Fatalf("Refresh: %v", err)
	}
}

func TestAsyncPurgeDb(t *testing.T) {
	gdb, mock, raw := newMockDB(t)
	defer raw.Close()
	m, _ := NewManager(context.Background(), gdb)

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE `api_login_keys` SET `deleted_at`").
		WillReturnResult(sqlmock.NewResult(0, 3))
	mock.ExpectCommit()

	if err := m.AsyncPurgeDb(context.Background()); err != nil {
		t.Fatalf("AsyncPurgeDb: %v", err)
	}
}
