// Copyright 2021 Google LLC
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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/boozt-platform/ipam-autopilot/provider/ipam/config"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/idtoken"
)

func ResourceIpRange() *schema.Resource {
	return &schema.Resource{
		Create: resourceCreate,
		Read:   resourceRead,
		//Update: resourceUpdate,
		Delete: resourceDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"range_size": {
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
			},
			"parent": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"domain": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"cidr": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
			"labels": {
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}
func resourceCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(config.Config)
	range_size := d.Get("range_size").(int)
	parent := d.Get("parent").(string)
	name := d.Get("name").(string)
	domain := d.Get("domain").(string)
	cidr := d.Get("cidr").(string)
	labels := d.Get("labels").(map[string]interface{})
	url := fmt.Sprintf("%s/ranges", config.Url)
	var postBody []byte
	var err error
	body := map[string]interface{}{
		"range_size": range_size,
		"name":       name,
		"domain":     domain,
		"cidr":       cidr,
	}
	if parent != "" {
		body["parent"] = parent
	}
	if len(labels) > 0 {
		labelStrings := make(map[string]string, len(labels))
		for k, v := range labels {
			labelStrings[k] = v.(string)
		}
		body["labels"] = labelStrings
	}
	postBody, err = json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed marshalling json: %v", err)
	}
	fmt.Printf("%s", string(postBody))
	responseBody := bytes.NewBuffer(postBody)
	accessToken, err := getIdentityToken()
	if err != nil {
		return fmt.Errorf("unable to retrieve access token: %v", err)
	}
	req, err := http.NewRequest("POST", url, responseBody)
	if err != nil {
		return fmt.Errorf("failed creating request: %v", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed creating range: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == 200 {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("unable to read response: %v", err)
		}
		response := map[string]interface{}{}
		err = json.Unmarshal(body, &response)
		if err != nil {
			return fmt.Errorf("unable to unmarshal response body: %v", err)
		}
		d.SetId(fmt.Sprintf("%d", int(response["id"].(float64))))
		_ = d.Set("cidr", response["cidr"].(string))
		return nil
	} else {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed creating range status_code=%d, status=%s", resp.StatusCode, resp.Status)
		}

		return fmt.Errorf("failed creating range status_code=%d, status=%s,body=%s", resp.StatusCode, resp.Status, string(body))
	}
}

func resourceRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(config.Config)
	url := fmt.Sprintf("%s/ranges/%s", config.Url, d.Id())

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed creating request: %v", err)
	}
	accessToken, err := getIdentityToken()
	if err != nil {
		return fmt.Errorf("unable to retrieve access token: %v", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed querying range: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == 200 {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("unable to read response: %v", err)
		}
		response := map[string]interface{}{}
		err = json.Unmarshal(body, &response)
		if err != nil {
			return fmt.Errorf("unable to unmarshal response body: %v", err)
		}
		d.SetId(fmt.Sprintf("%d", int(response["id"].(float64))))
		_ = d.Set("cidr", response["cidr"].(string))
		if labelsRaw, ok := response["labels"]; ok && labelsRaw != nil {
			if labelsMap, ok := labelsRaw.(map[string]interface{}); ok {
				labels := make(map[string]string, len(labelsMap))
				for k, v := range labelsMap {
					labels[k] = fmt.Sprintf("%v", v)
				}
				_ = d.Set("labels", labels)
			}
		}
		return nil
	} else {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed reading range status_code=%d, status=%s", resp.StatusCode, resp.Status)
		}

		return fmt.Errorf("failed reading range status_code=%d, status=%s,body=%s", resp.StatusCode, resp.Status, string(body))
	}
}

func resourceDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(config.Config)

	url := fmt.Sprintf("%s/ranges/%s", config.Url, d.Id())

	client := &http.Client{}
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed creating release request: %v", err)
	}
	accessToken, err := getIdentityToken()
	if err != nil {
		return fmt.Errorf("unable to retrieve access token: %v", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed releasing range: %v", err)
	}
	if resp.StatusCode == 200 {
		return nil
	} else {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed releasing range status_code=%d, status=%s", resp.StatusCode, resp.Status)
		}

		return fmt.Errorf("failed releasing range status_code=%d, status=%s,body=%s", resp.StatusCode, resp.Status, string(body))
	}
}

func getIdentityToken() (string, error) {
	if os.Getenv("GCP_IDENTITY_TOKEN") != "" {
		return os.Getenv("GCP_IDENTITY_TOKEN"), nil
	}

	ctx := context.Background()
	audience := "http://ipam-autopilot.com"
	ts, err := idtoken.NewTokenSource(ctx, audience)
	if err != nil {
		if err.Error() != `idtoken: credential must be service_account, found "authorized_user"` {
			return "", err
		}
		gts, err := google.DefaultTokenSource(ctx)
		if err != nil {
			return "", err
		}
		token, err := gts.Token()
		if err != nil {
			return "", err
		}
		identityToken := token.Extra("id_token").(string)
		return identityToken, nil
	}
	token, err := ts.Token()
	if err != nil {
		return "", err
	}
	return token.AccessToken, nil
}
