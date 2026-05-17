package mysql

import (
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Initialize() opens a real MySQL connection via DSN — not exercised here.
// Integration testing it requires an actual MySQL/MariaDB instance.

// GORM's HasColumn issues two preamble queries before the actual lookup:
// SELECT DATABASE() and a SCHEMA_NAME lookup. Expect them so we can match
// the real check we care about.
func expectHasColumnPreamble(mock sqlmock.Sqlmock) {
	mock.ExpectQuery("SELECT DATABASE\\(\\)").
		WillReturnRows(sqlmock.NewRows([]string{"DATABASE()"}).AddRow("carbon_test"))
	mock.ExpectQuery("SELECT SCHEMA_NAME").
		WillReturnRows(sqlmock.NewRows([]string{"SCHEMA_NAME"}).AddRow("carbon_test"))
}

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

func TestDropLegacyApiKeyRoleColumn_NoColumn(t *testing.T) {
	gdb, mock, raw := newMockDB(t)
	defer raw.Close()

	// GORM's HasColumn issues an information_schema lookup. Return 0 rows
	// to indicate the column is already gone — function should no-op.
	expectHasColumnPreamble(mock)
	mock.ExpectQuery("(?i)SELECT count\\(\\*\\) FROM INFORMATION_SCHEMA\\.columns").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	if err := dropLegacyApiKeyRoleColumn(gdb); err != nil {
		t.Fatalf("dropLegacyApiKeyRoleColumn: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestDropLegacyApiKeyRoleColumn_DropsWhenPresent(t *testing.T) {
	gdb, mock, raw := newMockDB(t)
	defer raw.Close()

	// HasColumn returns 1 → column exists; expect an ALTER TABLE DROP COLUMN.
	expectHasColumnPreamble(mock)
	mock.ExpectQuery("(?i)SELECT count\\(\\*\\) FROM INFORMATION_SCHEMA\\.columns").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectExec("ALTER TABLE `api_keys` DROP COLUMN `role`").
		WillReturnResult(sqlmock.NewResult(0, 0))

	if err := dropLegacyApiKeyRoleColumn(gdb); err != nil {
		t.Fatalf("dropLegacyApiKeyRoleColumn: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestDropLegacyServerApiKeyColumn_NoColumn(t *testing.T) {
	gdb, mock, raw := newMockDB(t)
	defer raw.Close()

	expectHasColumnPreamble(mock)
	mock.ExpectQuery("(?i)SELECT count\\(\\*\\) FROM INFORMATION_SCHEMA\\.columns").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	if err := dropLegacyServerApiKeyColumn(gdb); err != nil {
		t.Fatalf("dropLegacyServerApiKeyColumn: %v", err)
	}
}

func TestDropLegacyServerApiKeyColumn_DropsWhenPresent(t *testing.T) {
	gdb, mock, raw := newMockDB(t)
	defer raw.Close()

	expectHasColumnPreamble(mock)
	mock.ExpectQuery("(?i)SELECT count\\(\\*\\) FROM INFORMATION_SCHEMA\\.columns").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectExec("ALTER TABLE `servers` DROP COLUMN `api_key_id`").
		WillReturnResult(sqlmock.NewResult(0, 0))

	if err := dropLegacyServerApiKeyColumn(gdb); err != nil {
		t.Fatalf("dropLegacyServerApiKeyColumn: %v", err)
	}
}
