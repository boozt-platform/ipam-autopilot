// Copyright 2026 Boozt Fashion AB
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

//go:build integration

package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/go-sql-driver/mysql"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	tcmysql "github.com/testcontainers/testcontainers-go/modules/mysql"
)

func setupTestDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()
	ctx := context.Background()

	container, err := tcmysql.Run(ctx,
		"mysql:8.4",
		tcmysql.WithDatabase("ipam"),
		tcmysql.WithUsername("ipam"),
		tcmysql.WithPassword("ipam"),
	)
	require.NoError(t, err)

	host, err := container.Host(ctx)
	require.NoError(t, err)
	port, err := container.MappedPort(ctx, "3306")
	require.NoError(t, err)

	cfg := mysql.Config{
		User:                 "ipam",
		Passwd:               "ipam",
		Net:                  "tcp",
		Addr:                 fmt.Sprintf("%s:%s", host, port.Port()),
		DBName:               "ipam",
		MultiStatements:      true,
		AllowNativePasswords: true,
	}

	database, err := sql.Open("mysql", cfg.FormatDSN())
	require.NoError(t, err)

	err = MigrateDatabase("ipam", database)
	require.NoError(t, err)

	cleanup := func() {
		database.Close()
		container.Terminate(ctx)
	}
	return database, cleanup
}

func doRequest(app *fiber.App, method, path string, body interface{}) (*httptest.ResponseRecorder, []byte) {
	var reqBody io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		reqBody = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, path, reqBody)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, _ := app.Test(req, 10000)
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	return nil, respBody
}

func doRequestWithStatus(app *fiber.App, method, path string, body interface{}) (int, []byte) {
	var reqBody io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		reqBody = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, path, reqBody)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, _ := app.Test(req, 10000)
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, respBody
}

// --- Domain tests ---

func TestCreateDomain(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()
	app := newApp(database)

	status, body := doRequestWithStatus(app, "POST", "/api/v1/domains", map[string]interface{}{
		"name": "test-domain",
		"vpcs": []string{"projects/test/networks/vpc1"},
	})
	assert.Equal(t, 200, status)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(body, &resp))
	assert.NotNil(t, resp["id"])
}

func TestGetDomains(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()
	app := newApp(database)

	// Create one first
	doRequest(app, "POST", "/api/v1/domains", map[string]interface{}{
		"name": "test-domain",
		"vpcs": []string{},
	})

	status, body := doRequestWithStatus(app, "GET", "/api/v1/domains", nil)
	assert.Equal(t, 200, status)

	var resp []interface{}
	require.NoError(t, json.Unmarshal(body, &resp))
	assert.Len(t, resp, 1)
}

func TestGetDomain(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()
	app := newApp(database)

	_, createBody := doRequest(app, "POST", "/api/v1/domains", map[string]interface{}{
		"name": "test-domain",
		"vpcs": []string{},
	})
	var created map[string]interface{}
	require.NoError(t, json.Unmarshal(createBody, &created))
	id := int(created["id"].(float64))

	status, body := doRequestWithStatus(app, "GET", fmt.Sprintf("/api/v1/domains/%d", id), nil)
	assert.Equal(t, 200, status)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(body, &resp))
	assert.Equal(t, "test-domain", resp["name"])
}

func TestUpdateDomain(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()
	app := newApp(database)

	_, createBody := doRequest(app, "POST", "/api/v1/domains", map[string]interface{}{
		"name": "original",
		"vpcs": []string{},
	})
	var created map[string]interface{}
	require.NoError(t, json.Unmarshal(createBody, &created))
	id := int(created["id"].(float64))

	status, _ := doRequestWithStatus(app, "PUT", fmt.Sprintf("/api/v1/domains/%d", id), map[string]interface{}{
		"name": "updated",
	})
	assert.Equal(t, 200, status)

	_, getBody := doRequest(app, "GET", fmt.Sprintf("/api/v1/domains/%d", id), nil)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(getBody, &resp))
	assert.Equal(t, "updated", resp["name"])
}

func TestDeleteDomain(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()
	app := newApp(database)

	_, createBody := doRequest(app, "POST", "/api/v1/domains", map[string]interface{}{
		"name": "to-delete",
		"vpcs": []string{},
	})
	var created map[string]interface{}
	require.NoError(t, json.Unmarshal(createBody, &created))
	id := int(created["id"].(float64))

	status, _ := doRequestWithStatus(app, "DELETE", fmt.Sprintf("/api/v1/domains/%d", id), nil)
	assert.Equal(t, 200, status)
}

// --- Range tests ---

func setupDomainAndParent(t *testing.T, app *fiber.App) (domainID int, parentID int) {
	t.Helper()

	_, domainBody := doRequest(app, "POST", "/api/v1/domains", map[string]interface{}{
		"name": "test-domain",
		"vpcs": []string{},
	})
	var domain map[string]interface{}
	require.NoError(t, json.Unmarshal(domainBody, &domain))
	domainID = int(domain["id"].(float64))

	// Insert a parent range directly (cidr + domain)
	_, parentBody := doRequest(app, "POST", "/api/v1/ranges", map[string]interface{}{
		"name":   "parent",
		"cidr":   "10.0.0.0/8",
		"domain": fmt.Sprintf("%d", domainID),
	})
	var parent map[string]interface{}
	require.NoError(t, json.Unmarshal(parentBody, &parent))
	parentID = int(parent["id"].(float64))

	return domainID, parentID
}

func TestCreateRange_AutoAllocate(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()
	app := newApp(database)

	domainID, parentID := setupDomainAndParent(t, app)

	status, body := doRequestWithStatus(app, "POST", "/api/v1/ranges", map[string]interface{}{
		"name":       "gke-nodes",
		"range_size": 22,
		"parent":     fmt.Sprintf("%d", parentID),
		"domain":     fmt.Sprintf("%d", domainID),
	})
	assert.Equal(t, 200, status)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(body, &resp))
	assert.NotEmpty(t, resp["cidr"])
	assert.NotNil(t, resp["id"])
}

func TestCreateRange_DirectCidr(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()
	app := newApp(database)

	domainID, _ := setupDomainAndParent(t, app)

	status, body := doRequestWithStatus(app, "POST", "/api/v1/ranges", map[string]interface{}{
		"name":   "specific-range",
		"cidr":   "10.1.0.0/24",
		"domain": fmt.Sprintf("%d", domainID),
	})
	assert.Equal(t, 200, status)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(body, &resp))
	assert.Equal(t, "10.1.0.0/24", resp["cidr"])
}

func TestGetRanges(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()
	app := newApp(database)

	status, body := doRequestWithStatus(app, "GET", "/api/v1/ranges", nil)
	assert.Equal(t, 200, status)

	// Should return empty array or null (no ranges yet)
	_ = body
}

func TestGetRange(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()
	app := newApp(database)

	domainID, _ := setupDomainAndParent(t, app)

	_, createBody := doRequest(app, "POST", "/api/v1/ranges", map[string]interface{}{
		"name":   "my-range",
		"cidr":   "10.2.0.0/24",
		"domain": fmt.Sprintf("%d", domainID),
	})
	var created map[string]interface{}
	require.NoError(t, json.Unmarshal(createBody, &created))
	id := int(created["id"].(float64))

	status, body := doRequestWithStatus(app, "GET", fmt.Sprintf("/api/v1/ranges/%d", id), nil)
	assert.Equal(t, 200, status)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(body, &resp))
	assert.Equal(t, "10.2.0.0/24", resp["cidr"])
	assert.Equal(t, "my-range", resp["name"])
}

func TestDeleteRange(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()
	app := newApp(database)

	domainID, _ := setupDomainAndParent(t, app)

	_, createBody := doRequest(app, "POST", "/api/v1/ranges", map[string]interface{}{
		"name":   "to-delete",
		"cidr":   "10.3.0.0/24",
		"domain": fmt.Sprintf("%d", domainID),
	})
	var created map[string]interface{}
	require.NoError(t, json.Unmarshal(createBody, &created))
	id := int(created["id"].(float64))

	status, _ := doRequestWithStatus(app, "DELETE", fmt.Sprintf("/api/v1/ranges/%d", id), nil)
	assert.Equal(t, 200, status)
}

// --- Legacy route backward compat ---

func TestLegacyRoutesStillWork(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()
	app := newApp(database)

	// Old /domains path should still work (for Terraform provider compat)
	status, _ := doRequestWithStatus(app, "GET", "/domains", nil)
	assert.Equal(t, 200, status)

	status, _ = doRequestWithStatus(app, "GET", "/ranges", nil)
	assert.Equal(t, 200, status)
}
