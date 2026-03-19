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
	"database/sql"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/boozt-platform/ipam-autopilot/container/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// seedCAISubnets inserts rows directly into cai_subnets, simulating a CAI sync.
func seedCAISubnets(t *testing.T, db *sql.DB, network string, cidrs []string) {
	t.Helper()
	for _, cidr := range cidrs {
		_, err := db.Exec(
			`INSERT INTO cai_subnets (network, cidr, last_synced_at) VALUES (?, ?, NOW())
			 ON DUPLICATE KEY UPDATE last_synced_at = NOW()`,
			network, cidr)
		require.NoError(t, err)
	}
}

// TestCAI_AllocationAvoidsSeededSubnets verifies that auto-allocation skips CIDRs
// already present in cai_subnets for the domain's VPC.
func TestCAI_AllocationAvoidsSeededSubnets(t *testing.T) {
	t.Setenv("IPAM_CAI_ORG_ID", "fake-org")
	t.Setenv("IPAM_CAI_DB_SYNC", "TRUE")

	database, cleanup := setupTestDB(t)
	defer cleanup()
	app := server.NewApp(database)

	vpcURL := "https://www.googleapis.com/compute/v1/projects/test-proj/global/networks/test-vpc"

	// Create domain with VPC
	_, domainBody := doRequest(app, "POST", "/api/v1/domains", map[string]interface{}{
		"name": "cai-test-domain",
		"vpcs": []string{vpcURL},
	})
	var domain map[string]interface{}
	require.NoError(t, json.Unmarshal(domainBody, &domain))
	domainID := int(domain["id"].(float64))

	// Create parent range
	_, parentBody := doRequest(app, "POST", "/api/v1/ranges", map[string]interface{}{
		"name":   "parent",
		"cidr":   "10.60.0.0/16",
		"domain": fmt.Sprintf("%d", domainID),
	})
	var parent map[string]interface{}
	require.NoError(t, json.Unmarshal(parentBody, &parent))
	parentID := int(parent["id"].(float64))

	// Seed cai_subnets: occupy first three /24s as if they exist in GCP
	seedCAISubnets(t, database, vpcURL, []string{
		"10.60.0.0/24",
		"10.60.1.0/24",
		"10.60.2.0/24",
	})

	// Auto-allocate — must skip the seeded CIDRs and give us 10.60.3.0/24
	status, body := doRequestWithStatus(app, "POST", "/api/v1/ranges", map[string]interface{}{
		"name":       "new-subnet",
		"range_size": 24,
		"parent":     fmt.Sprintf("%d", parentID),
		"domain":     fmt.Sprintf("%d", domainID),
	})
	assert.Equal(t, 200, status)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(body, &resp))
	assert.Equal(t, "10.60.3.0/24", resp["cidr"])
}

// TestCAI_ShortVpcNameMatches verifies that a routing domain storing only the short
// VPC name (e.g. "my-vpc") still matches full CAI network URLs.
func TestCAI_ShortVpcNameMatches(t *testing.T) {
	t.Setenv("IPAM_CAI_ORG_ID", "fake-org")
	t.Setenv("IPAM_CAI_DB_SYNC", "TRUE")

	database, cleanup := setupTestDB(t)
	defer cleanup()
	app := server.NewApp(database)

	// Domain stores only short VPC name
	_, domainBody := doRequest(app, "POST", "/api/v1/domains", map[string]interface{}{
		"name": "short-name-domain",
		"vpcs": []string{"test-vpc"},
	})
	var domain map[string]interface{}
	require.NoError(t, json.Unmarshal(domainBody, &domain))
	domainID := int(domain["id"].(float64))

	_, parentBody := doRequest(app, "POST", "/api/v1/ranges", map[string]interface{}{
		"name":   "parent",
		"cidr":   "10.61.0.0/16",
		"domain": fmt.Sprintf("%d", domainID),
	})
	var parent map[string]interface{}
	require.NoError(t, json.Unmarshal(parentBody, &parent))
	parentID := int(parent["id"].(float64))

	// CAI stores full URL — short name in domain must still match
	fullURL := "https://www.googleapis.com/compute/v1/projects/test-proj/global/networks/test-vpc"
	seedCAISubnets(t, database, fullURL, []string{"10.61.0.0/24", "10.61.1.0/24"})

	status, body := doRequestWithStatus(app, "POST", "/api/v1/ranges", map[string]interface{}{
		"name":       "new-subnet",
		"range_size": 24,
		"parent":     fmt.Sprintf("%d", parentID),
		"domain":     fmt.Sprintf("%d", domainID),
	})
	assert.Equal(t, 200, status)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(body, &resp))
	assert.Equal(t, "10.61.2.0/24", resp["cidr"])
}

// TestCAI_DifferentVpcNotAffected verifies that subnets from a different VPC
// do not block allocation in an unrelated domain.
func TestCAI_DifferentVpcNotAffected(t *testing.T) {
	t.Setenv("IPAM_CAI_ORG_ID", "fake-org")
	t.Setenv("IPAM_CAI_DB_SYNC", "TRUE")

	database, cleanup := setupTestDB(t)
	defer cleanup()
	app := server.NewApp(database)

	_, domainBody := doRequest(app, "POST", "/api/v1/domains", map[string]interface{}{
		"name": "isolated-domain",
		"vpcs": []string{"my-vpc"},
	})
	var domain map[string]interface{}
	require.NoError(t, json.Unmarshal(domainBody, &domain))
	domainID := int(domain["id"].(float64))

	_, parentBody := doRequest(app, "POST", "/api/v1/ranges", map[string]interface{}{
		"name":   "parent",
		"cidr":   "10.62.0.0/16",
		"domain": fmt.Sprintf("%d", domainID),
	})
	var parent map[string]interface{}
	require.NoError(t, json.Unmarshal(parentBody, &parent))
	parentID := int(parent["id"].(float64))

	// Seed subnets for a completely different VPC — should not affect our domain
	otherVPC := "https://www.googleapis.com/compute/v1/projects/other-proj/global/networks/other-vpc"
	seedCAISubnets(t, database, otherVPC, []string{
		"10.62.0.0/24",
		"10.62.1.0/24",
		"10.62.2.0/24",
	})

	status, body := doRequestWithStatus(app, "POST", "/api/v1/ranges", map[string]interface{}{
		"name":       "new-subnet",
		"range_size": 24,
		"parent":     fmt.Sprintf("%d", parentID),
		"domain":     fmt.Sprintf("%d", domainID),
	})
	assert.Equal(t, 200, status)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(body, &resp))
	// First available since other-vpc subnets are ignored
	assert.Equal(t, "10.62.0.0/24", resp["cidr"])
}
