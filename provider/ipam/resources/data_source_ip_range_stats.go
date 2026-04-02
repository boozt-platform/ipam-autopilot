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

package resources

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/boozt-platform/ipam-autopilot/provider/ipam/config"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DataSourceIpRangeStats() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceStatsRead,
		Schema: map[string]*schema.Schema{
			"id": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "ID of the IP range. Use this when the range is managed in the same stack to avoid stale name lookups.",
			},
			"name": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Name of the IP range to look up. Use `id` when the range is managed in the same stack.",
			},
			"cidr": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The CIDR block of the range.",
			},
			"total_addresses": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Total number of IP addresses in the range.",
			},
			"used_addresses": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Number of IP addresses allocated to direct child ranges.",
			},
			"free_addresses": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Number of IP addresses not yet allocated.",
			},
			"utilization_pct": {
				Type:        schema.TypeFloat,
				Computed:    true,
				Description: "Percentage of address space allocated to direct child ranges (0-100).",
			},
		},
	}
}

func dataSourceStatsRead(d *schema.ResourceData, meta interface{}) error {
	cfg := meta.(config.Config)

	accessToken, err := getIdentityToken(cfg.Url)
	if err != nil {
		return fmt.Errorf("unable to retrieve access token: %v", err)
	}

	rangeID := d.Get("id").(string)
	if rangeID == "" {
		rangeID = d.Id()
	}

	if rangeID == "" {
		name, ok := d.GetOk("name")
		if !ok || name.(string) == "" {
			return fmt.Errorf("one of `id` or `name` must be set")
		}
		rangeID, err = resolveRangeIDByName(cfg.Url, name.(string), accessToken)
		if err != nil {
			return err
		}
	}

	// Fetch full range including stats.
	getURL := fmt.Sprintf("%s/api/v1/ranges/%s", cfg.Url, rangeID)
	getReq, err := http.NewRequest("GET", getURL, nil)
	if err != nil {
		return fmt.Errorf("failed creating request: %v", err)
	}
	getReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	client := &http.Client{}
	getResp, err := client.Do(getReq)
	if err != nil {
		return fmt.Errorf("failed fetching range stats: %v", err)
	}
	defer getResp.Body.Close() //nolint:errcheck

	getBody, err := io.ReadAll(getResp.Body)
	if err != nil {
		return fmt.Errorf("unable to read response: %v", err)
	}
	if getResp.StatusCode != 200 {
		return fmt.Errorf("failed fetching range status_code=%d, body=%s", getResp.StatusCode, string(getBody))
	}

	var rang map[string]interface{}
	if err := json.Unmarshal(getBody, &rang); err != nil {
		return fmt.Errorf("unable to unmarshal response: %v", err)
	}

	d.SetId(rangeID)
	_ = d.Set("id", rangeID)
	_ = d.Set("cidr", rang["cidr"].(string))

	if nameVal, ok := rang["name"]; ok && nameVal != nil {
		_ = d.Set("name", nameVal.(string))
	}

	if statsRaw, ok := rang["stats"]; ok && statsRaw != nil {
		stats := statsRaw.(map[string]interface{})
		_ = d.Set("total_addresses", int(stats["total_addresses"].(float64)))
		_ = d.Set("used_addresses", int(stats["used_addresses"].(float64)))
		_ = d.Set("free_addresses", int(stats["free_addresses"].(float64)))
		_ = d.Set("utilization_pct", stats["utilization_pct"].(float64))
	}

	return nil
}

func resolveRangeIDByName(baseURL, name, accessToken string) (string, error) {
	listURL := fmt.Sprintf("%s/ranges?name=%s", baseURL, name)
	req, err := http.NewRequest("GET", listURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed creating request: %v", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed querying ranges: %v", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("unable to read response: %v", err)
	}
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("failed querying ranges status_code=%d, body=%s", resp.StatusCode, string(body))
	}

	var results []map[string]interface{}
	if err := json.Unmarshal(body, &results); err != nil {
		return "", fmt.Errorf("unable to unmarshal response: %v", err)
	}
	if len(results) == 0 {
		return "", fmt.Errorf("no ip range found with name %q", name)
	}
	return fmt.Sprintf("%d", int(results[0]["id"].(float64))), nil
}
