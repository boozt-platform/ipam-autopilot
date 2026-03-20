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
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the IP range to look up.",
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
	name := d.Get("name").(string)

	url := fmt.Sprintf("%s/ranges?name=%s", cfg.Url, name)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed creating request: %v", err)
	}
	accessToken, err := getIdentityToken(cfg.Url)
	if err != nil {
		return fmt.Errorf("unable to retrieve access token: %v", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed querying ranges: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("unable to read response: %v", err)
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("failed querying ranges status_code=%d, body=%s", resp.StatusCode, string(body))
	}

	var results []map[string]interface{}
	if err := json.Unmarshal(body, &results); err != nil {
		return fmt.Errorf("unable to unmarshal response: %v", err)
	}
	if len(results) == 0 {
		return fmt.Errorf("no ip range found with name %q", name)
	}

	match := results[0]
	d.SetId(fmt.Sprintf("%d", int(match["id"].(float64))))
	_ = d.Set("cidr", match["cidr"].(string))

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
