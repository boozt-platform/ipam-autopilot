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
	"net"
	"sync"
	"testing"

	"github.com/boozt-platform/ipam-autopilot/container/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateRange_CIDRHostBitsRejected(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()
	app := server.NewApp(database)

	domainID, _ := setupDomainAndParent(t, app)

	status, body := doRequestWithStatus(app, "POST", "/api/v1/ranges", map[string]interface{}{
		"name":   "host-bits-set",
		"cidr":   "10.0.0.1/24",
		"domain": fmt.Sprintf("%d", domainID),
	})
	assert.Equal(t, 400, status)
	assert.Contains(t, string(body), "host bits set")
}

func TestCreateRange_IPv6CIDRRejected(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()
	app := server.NewApp(database)

	domainID, _ := setupDomainAndParent(t, app)

	status, body := doRequestWithStatus(app, "POST", "/api/v1/ranges", map[string]interface{}{
		"name":   "ipv6-range",
		"cidr":   "2001:db8::/32",
		"domain": fmt.Sprintf("%d", domainID),
	})
	assert.Equal(t, 400, status)
	assert.Contains(t, string(body), "IPv4")
}

func TestCreateRange_RangeSizeBounds(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()
	app := server.NewApp(database)

	domainID, parentID := setupDomainAndParent(t, app)

	for _, size := range []int{0, 33, -1} {
		status, _ := doRequestWithStatus(app, "POST", "/api/v1/ranges", map[string]interface{}{
			"name":       fmt.Sprintf("bad-size-%d", size),
			"range_size": size,
			"parent":     fmt.Sprintf("%d", parentID),
			"domain":     fmt.Sprintf("%d", domainID),
		})
		assert.Equal(t, 400, status, "expected 400 for range_size=%d", size)
	}
}

func TestCreateRange_WhitespaceNameRejected(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()
	app := server.NewApp(database)

	domainID, _ := setupDomainAndParent(t, app)

	status, body := doRequestWithStatus(app, "POST", "/api/v1/ranges", map[string]interface{}{
		"name":   "   ",
		"cidr":   "10.99.0.0/24",
		"domain": fmt.Sprintf("%d", domainID),
	})
	assert.Equal(t, 400, status)
	assert.Contains(t, string(body), "name")
}

func TestCreateRoutingDomain_EmptyNameRejected(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()
	app := server.NewApp(database)

	status, _ := doRequestWithStatus(app, "POST", "/api/v1/domains", map[string]interface{}{
		"name": "",
		"vpcs": []string{},
	})
	assert.Equal(t, 400, status)
}

func TestDeleteRange_WithChildrenRejected(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()
	app := server.NewApp(database)

	domainID, parentID := setupDomainAndParent(t, app)

	// Create a child range under the parent.
	_, childBody := doRequest(app, "POST", "/api/v1/ranges", map[string]interface{}{
		"name":       "child-range",
		"range_size": 22,
		"parent":     fmt.Sprintf("%d", parentID),
		"domain":     fmt.Sprintf("%d", domainID),
	})
	var child map[string]interface{}
	require.NoError(t, json.Unmarshal(childBody, &child))
	require.NotNil(t, child["id"], "child range creation failed: %s", string(childBody))

	// Deleting the parent should be rejected with 409.
	status, body := doRequestWithStatus(app, "DELETE", fmt.Sprintf("/api/v1/ranges/%d", parentID), nil)
	assert.Equal(t, 409, status)
	assert.Contains(t, string(body), "child")
}

func TestDeleteDomain_WithRangesRejected(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()
	app := server.NewApp(database)

	domainID, _ := setupDomainAndParent(t, app)

	// Deleting the domain should be rejected because it has ranges.
	status, body := doRequestWithStatus(app, "DELETE", fmt.Sprintf("/api/v1/domains/%d", domainID), nil)
	assert.Equal(t, 409, status)
	assert.Contains(t, string(body), "ranges")
}

func TestCreateRange_LabelControlCharRejected(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()
	app := server.NewApp(database)

	domainID, _ := setupDomainAndParent(t, app)

	status, body := doRequestWithStatus(app, "POST", "/api/v1/ranges", map[string]interface{}{
		"name":   "ctrl-char-label",
		"cidr":   "10.88.0.0/24",
		"domain": fmt.Sprintf("%d", domainID),
		"labels": map[string]string{
			"bad\nkey": "value",
		},
	})
	assert.Equal(t, 400, status)
	assert.Contains(t, string(body), "invalid characters")
}

func TestConcurrentAllocation(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()
	app := server.NewApp(database)

	domainID, parentID := setupDomainAndParent(t, app)

	const workers = 5
	cidrs := make([]string, workers)
	errs := make([]error, workers)

	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		i := i
		go func() {
			defer wg.Done()
			status, body := doRequestWithStatus(app, "POST", "/api/v1/ranges", map[string]interface{}{
				"name":       fmt.Sprintf("concurrent-%d", i),
				"range_size": 22,
				"parent":     fmt.Sprintf("%d", parentID),
				"domain":     fmt.Sprintf("%d", domainID),
			})
			if status != 200 {
				errs[i] = fmt.Errorf("worker %d got status %d: %s", i, status, string(body))
				return
			}
			var resp map[string]interface{}
			if err := json.Unmarshal(body, &resp); err != nil {
				errs[i] = fmt.Errorf("worker %d unmarshal error: %w", i, err)
				return
			}
			cidrs[i] = resp["cidr"].(string)
		}()
	}
	wg.Wait()

	for i, err := range errs {
		require.NoError(t, err, "worker %d failed", i)
	}

	// Verify all CIDRs are non-empty and non-overlapping.
	nets := make([]*net.IPNet, workers)
	for i, c := range cidrs {
		require.NotEmpty(t, c, "worker %d got empty cidr", i)
		_, ipNet, err := net.ParseCIDR(c)
		require.NoError(t, err, "worker %d cidr %q invalid", i, c)
		nets[i] = ipNet
	}

	for i := 0; i < workers; i++ {
		for j := i + 1; j < workers; j++ {
			assert.False(t, nets[i].Contains(nets[j].IP) || nets[j].Contains(nets[i].IP),
				"CIDRs overlap: %s and %s", nets[i], nets[j])
		}
	}
}
