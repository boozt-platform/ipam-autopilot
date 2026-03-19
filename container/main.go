// Copyright 2021 Google LLC
// Copyright 2026 Boozt Fashion AB (modifications)
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/boozt-platform/ipam-autopilot/container/server"
	"github.com/go-sql-driver/mysql"
)

func main() {
	ctx := context.Background()

	logger := server.InitLogger()
	slog.SetDefault(logger)

	shutdownTracer, err := server.InitTracer(ctx)
	if err != nil {
		slog.Error("failed to initialize tracer", "error", err)
		os.Exit(1)
	}
	defer shutdownTracer(ctx) //nolint:errcheck

	cfg := mysql.Config{
		User:                 os.Getenv("IPAM_DATABASE_USER"),
		Passwd:               os.Getenv("IPAM_DATABASE_PASSWORD"),
		Net:                  os.Getenv("IPAM_DATABASE_NET"),
		Addr:                 os.Getenv("IPAM_DATABASE_HOST"),
		DBName:               os.Getenv("IPAM_DATABASE_NAME"),
		MultiStatements:      true,
		AllowNativePasswords: true,
	}

	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		slog.Error("failed to open database", "error", err)
		os.Exit(1)
	}
	db.SetMaxOpenConns(100)
	db.SetMaxIdleConns(5)
	defer db.Close()

	if os.Getenv("IPAM_DISABLE_DATABASE_MIGRATION") != "TRUE" {
		if err = server.MigrateDatabase(os.Getenv("IPAM_DATABASE_NAME"), db); err != nil {
			slog.Error("failed to migrate database", "error", err)
			os.Exit(1)
		}
	}

	app := server.NewApp(db)

	if orgID := os.Getenv("IPAM_CAI_ORG_ID"); orgID != "" && os.Getenv("IPAM_CAI_DB_SYNC") == "TRUE" {
		interval := 5 * time.Minute
		if raw := os.Getenv("IPAM_CAI_SYNC_INTERVAL"); raw != "" {
			if d, err := time.ParseDuration(raw); err == nil {
				interval = d
			} else {
				slog.Warn("invalid IPAM_CAI_SYNC_INTERVAL, using default", "value", raw, "default", interval)
			}
		}
		if err := server.StartCAISyncLoop(ctx, orgID, interval); err != nil {
			slog.Error("CAI sync failed on startup", "error", err)
			os.Exit(1)
		}
	}

	port := int64(8080)
	if os.Getenv("IPAM_PORT") != "" {
		port, err = strconv.ParseInt(os.Getenv("IPAM_PORT"), 10, 64)
		if err != nil {
			slog.Error("failed to parse PORT", "value", os.Getenv("IPAM_PORT"), "error", err)
			os.Exit(1)
		}
	}

	slog.Info("starting server", "port", port)
	if err := app.Listen(fmt.Sprintf(":%d", port)); err != nil {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
}
