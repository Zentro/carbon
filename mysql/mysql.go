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

	if err := db.AutoMigrate(
		&domain.ApiLoginKey{}, // independent
	); err != nil {
		return nil, fmt.Errorf("auto migration failed: %w", err)
	}

	if err := db.AutoMigrate(
		&domain.Server{}, // references ApiKey
	); err != nil {
		return nil, fmt.Errorf("auto migration failed: %w", err)
	}

	if err := db.AutoMigrate(
		&domain.Client{}, // references Server
	); err != nil {
		return nil, fmt.Errorf("auto migration failed: %w", err)
	}

	slog.Info("database migrations completed")

	return db, nil
}
