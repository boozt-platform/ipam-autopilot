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

package server

import (
	"database/sql"
	"os"

	"github.com/gofiber/fiber/v2"
)

var db *sql.DB

func NewApp(database *sql.DB) *fiber.App {
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
	if os.Getenv("IPAM_DISABLE_BULK_IMPORT") != "TRUE" {
		v1.Post("/ranges/import", ImportRanges)
	}
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
