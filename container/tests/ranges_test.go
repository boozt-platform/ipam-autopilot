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

package tests

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/boozt-platform/ipam-autopilot/container/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateRange_AutoAllocate(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()
	app := server.NewApp(database)

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
	app := server.NewApp(database)

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
	app := server.NewApp(database)

	status, _ := doRequestWithStatus(app, "GET", "/api/v1/ranges", nil)
	assert.Equal(t, 200, status)
}

func TestGetRange(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()
	app := server.NewApp(database)

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
	app := server.NewApp(database)

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

func TestCreateRange_WithLabels(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()
	app := server.NewApp(database)

	domainID, parentID := setupDomainAndParent(t, app)

	status, body := doRequestWithStatus(app, "POST", "/api/v1/ranges", map[string]interface{}{
		"name":       "gke-nodes-prod",
		"range_size": 22,
		"parent":     fmt.Sprintf("%d", parentID),
		"domain":     fmt.Sprintf("%d", domainID),
		"labels": map[string]string{
			"env":     "prod",
			"team":    "platform",
			"purpose": "gke-nodes",
		},
	})
	assert.Equal(t, 200, status)

	var created map[string]interface{}
	require.NoError(t, json.Unmarshal(body, &created))
	id := int(created["id"].(float64))

	status, body = doRequestWithStatus(app, "GET", fmt.Sprintf("/api/v1/ranges/%d", id), nil)
	assert.Equal(t, 200, status)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(body, &resp))
	labels, ok := resp["labels"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "prod", labels["env"])
	assert.Equal(t, "platform", labels["team"])
	assert.Equal(t, "gke-nodes", labels["purpose"])
}

func TestGetRanges_FilterByName(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()
	app := server.NewApp(database)

	domainID, _ := setupDomainAndParent(t, app)

	doRequest(app, "POST", "/api/v1/ranges", map[string]interface{}{
		"name":   "alpha",
		"cidr":   "10.10.0.0/24",
		"domain": fmt.Sprintf("%d", domainID),
	})
	doRequest(app, "POST", "/api/v1/ranges", map[string]interface{}{
		"name":   "beta",
		"cidr":   "10.11.0.0/24",
		"domain": fmt.Sprintf("%d", domainID),
	})

	status, body := doRequestWithStatus(app, "GET", "/api/v1/ranges?name=alpha", nil)
	assert.Equal(t, 200, status)

	var resp []map[string]interface{}
	require.NoError(t, json.Unmarshal(body, &resp))
	require.Len(t, resp, 1)
	assert.Equal(t, "alpha", resp[0]["name"])
	assert.Equal(t, "10.10.0.0/24", resp[0]["cidr"])
}

func TestCreateRange_NameRequired(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()
	app := server.NewApp(database)

	domainID, _ := setupDomainAndParent(t, app)

	status, _ := doRequestWithStatus(app, "POST", "/api/v1/ranges", map[string]interface{}{
		"cidr":   "10.20.0.0/24",
		"domain": fmt.Sprintf("%d", domainID),
	})
	assert.Equal(t, 400, status)
}

func TestCreateRange_NameTooLong(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()
	app := server.NewApp(database)

	domainID, _ := setupDomainAndParent(t, app)

	status, body := doRequestWithStatus(app, "POST", "/api/v1/ranges", map[string]interface{}{
		"name":   string(make([]byte, 256)),
		"cidr":   "10.21.0.0/24",
		"domain": fmt.Sprintf("%d", domainID),
	})
	assert.Equal(t, 400, status)
	assert.Contains(t, string(body), "255")
}

func TestCreateRange_LabelKeyTooLong(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()
	app := server.NewApp(database)

	domainID, _ := setupDomainAndParent(t, app)

	status, _ := doRequestWithStatus(app, "POST", "/api/v1/ranges", map[string]interface{}{
		"name":   "valid-name",
		"cidr":   "10.22.0.0/24",
		"domain": fmt.Sprintf("%d", domainID),
		"labels": map[string]string{
			string(make([]byte, 64)): "value",
		},
	})
	assert.Equal(t, 400, status)
}

func TestCreateRange_LabelValueTooLong(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()
	app := server.NewApp(database)

	domainID, _ := setupDomainAndParent(t, app)

	status, _ := doRequestWithStatus(app, "POST", "/api/v1/ranges", map[string]interface{}{
		"name":   "valid-name",
		"cidr":   "10.23.0.0/24",
		"domain": fmt.Sprintf("%d", domainID),
		"labels": map[string]string{
			"key": string(make([]byte, 256)),
		},
	})
	assert.Equal(t, 400, status)
}

func TestImportRanges_Basic(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()
	app := server.NewApp(database)

	domainID, _ := setupDomainAndParent(t, app)

	status, body := doRequestWithStatus(app, "POST", "/api/v1/ranges/import", []map[string]interface{}{
		{"name": "legacy-a", "cidr": "10.50.0.0/24", "domain": fmt.Sprintf("%d", domainID)},
		{"name": "legacy-b", "cidr": "10.50.1.0/24", "domain": fmt.Sprintf("%d", domainID)},
	})
	assert.Equal(t, 200, status)

	var result map[string]interface{}
	require.NoError(t, json.Unmarshal(body, &result))
	assert.Equal(t, float64(2), result["imported"])
	assert.Equal(t, float64(0), result["skipped"])
	assert.Empty(t, result["errors"])
}

func TestImportRanges_Idempotent(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()
	app := server.NewApp(database)

	domainID, _ := setupDomainAndParent(t, app)

	payload := []map[string]interface{}{
		{"name": "legacy-c", "cidr": "10.51.0.0/24", "domain": fmt.Sprintf("%d", domainID)},
	}

	doRequest(app, "POST", "/api/v1/ranges/import", payload)

	status, body := doRequestWithStatus(app, "POST", "/api/v1/ranges/import", payload)
	assert.Equal(t, 200, status)

	var result map[string]interface{}
	require.NoError(t, json.Unmarshal(body, &result))
	assert.Equal(t, float64(0), result["imported"])
	assert.Equal(t, float64(1), result["skipped"])
}

func TestImportRanges_WithLabels(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()
	app := server.NewApp(database)

	domainID, _ := setupDomainAndParent(t, app)

	status, _ := doRequestWithStatus(app, "POST", "/api/v1/ranges/import", []map[string]interface{}{
		{
			"name":   "legacy-d",
			"cidr":   "10.52.0.0/24",
			"domain": fmt.Sprintf("%d", domainID),
			"labels": map[string]string{"env": "prod"},
		},
	})
	assert.Equal(t, 200, status)

	_, listBody := doRequest(app, "GET", "/api/v1/ranges?name=legacy-d", nil)
	var ranges []map[string]interface{}
	require.NoError(t, json.Unmarshal(listBody, &ranges))
	require.Len(t, ranges, 1)
	labels := ranges[0]["labels"].(map[string]interface{})
	assert.Equal(t, "prod", labels["env"])
}

func TestImportRanges_ValidationErrors(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()
	app := server.NewApp(database)

	domainID, _ := setupDomainAndParent(t, app)

	status, body := doRequestWithStatus(app, "POST", "/api/v1/ranges/import", []map[string]interface{}{
		{"name": "", "cidr": "10.53.0.0/24", "domain": fmt.Sprintf("%d", domainID)},
		{"name": "valid", "cidr": "", "domain": fmt.Sprintf("%d", domainID)},
		{"name": "ok", "cidr": "10.53.1.0/24", "domain": fmt.Sprintf("%d", domainID)},
	})
	assert.Equal(t, 200, status)

	var result map[string]interface{}
	require.NoError(t, json.Unmarshal(body, &result))
	assert.Equal(t, float64(1), result["imported"])
	assert.Len(t, result["errors"].([]interface{}), 2)
}
