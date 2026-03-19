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

	"github.com/go-sql-driver/mysql"
	"github.com/gofiber/fiber/v2"
)

var db *sql.DB

func newApp(database *sql.DB) *fiber.App {
	db = database

	app := fiber.New()
	app.Use(requestIDMiddleware())
	app.Use(tracingMiddleware())
	app.Use(accessLogMiddleware())

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("IPAM Autopilot up and running!")
	})

	// Terraform provider registry — versioned by terraform protocol, kept as-is
	app.Get("/.well-known/terraform.json", GetTerraformDiscovery)
	app.Get("/terraform/providers/v1/ipam-autopilot/ipam/versions", GetTerraformVersions)
	app.Get("/terraform/providers/v1/ipam-autopilot/ipam/:version/download/:os/:arch", GetTerraformVersionDownload)

	// API v1
	v1 := app.Group("/api/v1")
	v1.Post("/ranges", CreateNewRange)
	v1.Get("/ranges", GetRanges)
	v1.Get("/ranges/:id", GetRange)
	v1.Delete("/ranges/:id", DeleteRange)

	v1.Get("/domains", GetRoutingDomains)
	v1.Get("/domains/:id", GetRoutingDomain)
	v1.Put("/domains/:id", UpdateRoutingDomain)
	v1.Post("/domains", CreateRoutingDomain)
	v1.Delete("/domains/:id", DeleteRoutingDomain)

	v1.Get("/audit", GetAuditLogs)

	// Legacy routes — kept for Terraform provider backward compatibility
	// TODO: remove after provider is updated to use /api/v1
	app.Post("/ranges", CreateNewRange)
	app.Get("/ranges", GetRanges)
	app.Get("/ranges/:id", GetRange)
	app.Delete("/ranges/:id", DeleteRange)

	app.Get("/domains", GetRoutingDomains)
	app.Get("/domains/:id", GetRoutingDomain)
	app.Put("/domains/:id", UpdateRoutingDomain)
	app.Post("/domains", CreateRoutingDomain)
	app.Delete("/domains/:id", DeleteRoutingDomain)

	return app
}

func main() {
	ctx := context.Background()

	logger := initLogger()
	slog.SetDefault(logger)

	shutdownTracer, err := initTracer(ctx)
	if err != nil {
		slog.Error("failed to initialize tracer", "error", err)
		os.Exit(1)
	}
	defer shutdownTracer(ctx) //nolint:errcheck

	cfg := mysql.Config{
		User:                 os.Getenv("DATABASE_USER"),
		Passwd:               os.Getenv("DATABASE_PASSWORD"),
		Net:                  os.Getenv("DATABASE_NET"),
		Addr:                 os.Getenv("DATABASE_HOST"),
		DBName:               os.Getenv("DATABASE_NAME"),
		MultiStatements:      true,
		AllowNativePasswords: true,
	}

	db, err = sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		slog.Error("failed to open database", "error", err)
		os.Exit(1)
	}
	db.SetMaxOpenConns(100)
	db.SetMaxIdleConns(5)
	defer db.Close()

	if os.Getenv("DISABLE_DATABASE_MIGRATION") != "TRUE" {
		if err = MigrateDatabase(os.Getenv("DATABASE_NAME"), db); err != nil {
			slog.Error("failed to migrate database", "error", err)
			os.Exit(1)
		}
	}

	app := newApp(db)

	port := int64(8080)
	if os.Getenv("PORT") != "" {
		port, err = strconv.ParseInt(os.Getenv("PORT"), 10, 64)
		if err != nil {
			slog.Error("failed to parse PORT", "value", os.Getenv("PORT"), "error", err)
			os.Exit(1)
		}
	}

	slog.Info("starting server", "port", port)
	if err := app.Listen(fmt.Sprintf(":%d", port)); err != nil {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
}
