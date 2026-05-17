// Copyright (C) 2024, 2025 Rafael Galvan

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package mysql

import (
	"carbon/config"
	"carbon/domain"
	"fmt"
	"log/slog"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Initialize sets up the MySQL connection and performs AutoMigrate.
func Initialize() (*gorm.DB, error) {
	// Load DB config
	dbConf := config.Get().Db

	// Build DSN
	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s)/%s?charset=%s&parseTime=true&loc=Local",
		dbConf.Username,
		dbConf.Password,
		dbConf.Host,
		dbConf.Database,
		dbConf.Charset,
	)

	// Open GORM connection
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Error), // Only log errors
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MySQL: %w", err)
	}

	slog.Info("connected to the database")

	// AutoMigrate tables in order of dependencies
	// 1. Tables without foreign keys first
	// 2. Tables that reference others next
	if err := db.AutoMigrate(
		&domain.ApiKey{}, // independent
	); err != nil {
		return nil, fmt.Errorf("auto migration failed: %w", err)
	}

	if err := dropLegacyApiKeyRoleColumn(db); err != nil {
		return nil, fmt.Errorf("api_keys schema cleanup failed: %w", err)
	}

	if err := db.AutoMigrate(
		&domain.ApiLoginKey{}, // independent
	); err != nil {
		return nil, fmt.Errorf("auto migration failed: %w", err)
	}

	if err := db.AutoMigrate(
		&domain.Server{},
	); err != nil {
		return nil, fmt.Errorf("auto migration failed: %w", err)
	}

	if err := dropLegacyServerApiKeyColumn(db); err != nil {
		return nil, fmt.Errorf("server schema cleanup failed: %w", err)
	}

	if err := db.AutoMigrate(
		&domain.Client{}, // references Server
	); err != nil {
		return nil, fmt.Errorf("auto migration failed: %w", err)
	}

	slog.Info("database migrations completed")

	return db, nil
}

// dropLegacyServerApiKeyColumn removes the now-unused servers.api_key_id
// column and its associated unique index. Pre-bound-key schemas had a 1:1
// server-to-key relationship; bound API keys live on the api_keys side
// now (via target_kind/target_id). AutoMigrate doesn't drop columns on
// its own, so we ask explicitly. Idempotent.
func dropLegacyServerApiKeyColumn(db *gorm.DB) error {
	mig := db.Migrator()
	if !mig.HasColumn(&domain.Server{}, "api_key_id") {
		return nil
	}
	// MySQL drops associated indexes when the column goes; no explicit
	// DropIndex call needed.
	if err := mig.DropColumn(&domain.Server{}, "api_key_id"); err != nil {
		return err
	}
	slog.Info("dropped legacy servers.api_key_id column")
	return nil
}

// dropLegacyApiKeyRoleColumn removes the now-unused api_keys.role column.
// Roles live on the User record (XF IsStaff -> admin); per-key authority
// lives in api_keys.scopes. Idempotent.
func dropLegacyApiKeyRoleColumn(db *gorm.DB) error {
	mig := db.Migrator()
	if !mig.HasColumn(&domain.ApiKey{}, "role") {
		return nil
	}
	if err := mig.DropColumn(&domain.ApiKey{}, "role"); err != nil {
		return err
	}
	slog.Info("dropped legacy api_keys.role column")
	return nil
}
