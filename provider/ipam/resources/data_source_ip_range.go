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

func DataSourceIpRange() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceRead,
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
				Description: "The allocated CIDR block (e.g. `10.0.4.0/22`).",
			},
			"labels": {
				Type:        schema.TypeMap,
				Computed:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "Labels attached to the range.",
			},
		},
	}
}

func dataSourceRead(d *schema.ResourceData, meta interface{}) error {
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

	// Fetch range by ID.
	getURL := fmt.Sprintf("%s/api/v1/ranges/%s", cfg.Url, rangeID)
	req, err := http.NewRequest("GET", getURL, nil)
	if err != nil {
		return fmt.Errorf("failed creating request: %v", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed querying range: %v", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("unable to read response: %v", err)
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("failed querying range status_code=%d, body=%s", resp.StatusCode, string(body))
	}

	var match map[string]interface{}
	if err := json.Unmarshal(body, &match); err != nil {
		return fmt.Errorf("unable to unmarshal response: %v", err)
	}

	d.SetId(rangeID)
	_ = d.Set("id", rangeID)
	_ = d.Set("cidr", match["cidr"].(string))

	if nameVal, ok := match["name"]; ok && nameVal != nil {
		_ = d.Set("name", nameVal.(string))
	}

	if labelsRaw, ok := match["labels"]; ok && labelsRaw != nil {
		if labelsMap, ok := labelsRaw.(map[string]interface{}); ok {
			labels := make(map[string]string, len(labelsMap))
			for k, v := range labelsMap {
				labels[k] = fmt.Sprintf("%v", v)
			}
			_ = d.Set("labels", labels)
		}
	}

	return nil
}
