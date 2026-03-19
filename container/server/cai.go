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

package server

import (
	"context"
	"fmt"
	"strings"

	asset "cloud.google.com/go/asset/apiv1"
	assetpb "cloud.google.com/go/asset/apiv1/assetpb"
	"google.golang.org/api/iterator"
)

type caiSubnet struct {
	network string
	cidr    string
}

// fetchCAISubnets returns all Subnetwork assets under parent (e.g. "organizations/123").
// It does not filter by VPC — callers use vpcMatches to filter as needed.
func fetchCAISubnets(ctx context.Context, parent string) ([]caiSubnet, error) {
	client, err := asset.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	defer client.Close() //nolint:errcheck

	itr := client.ListAssets(ctx, &assetpb.ListAssetsRequest{
		Parent:      parent,
		AssetTypes:  []string{"compute.googleapis.com/Subnetwork"},
		ContentType: assetpb.ContentType_RESOURCE,
	})

	var subnets []caiSubnet
	for {
		a, err := itr.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		network := a.Resource.Data.Fields["network"].GetStringValue()
		cidr := a.Resource.Data.Fields["ipCidrRange"].GetStringValue()
		if network == "" || cidr == "" {
			continue
		}
		subnets = append(subnets, caiSubnet{network: network, cidr: cidr})

		// Also collect secondary ranges
		for _, raw := range a.Resource.Data.Fields["secondaryIpRanges"].GetListValue().AsSlice() {
			m, ok := raw.(map[string]interface{})
			if !ok {
				continue
			}
			if secondaryCidr, ok := m["ipCidrRange"].(string); ok && secondaryCidr != "" {
				subnets = append(subnets, caiSubnet{network: network, cidr: secondaryCidr})
			}
		}
	}
	return subnets, nil
}

// getLiveCAISubnetsForNetworks fetches subnets directly from the CAI API (no DB)
// and filters them by the given VPC identifiers. Used when IPAM_CAI_DB_SYNC is not enabled.
func getLiveCAISubnetsForNetworks(ctx context.Context, orgID string, networks []string) ([]Range, error) {
	subnets, err := fetchCAISubnets(ctx, fmt.Sprintf("organizations/%s", orgID))
	if err != nil {
		return nil, err
	}
	var result []Range
	for _, s := range subnets {
		for _, vpc := range networks {
			if vpcMatches(vpc, s.network) {
				result = append(result, Range{Cidr: s.cidr})
				break
			}
		}
	}
	return result, nil
}

// vpcMatches returns true if the storedVpc (from routing domain) matches a CAI network URL.
// Accepts both full URLs and short forms:
//
//	"https://www.googleapis.com/compute/v1/projects/my-proj/global/networks/my-vpc"
//	"projects/my-proj/global/networks/my-vpc"
//	"my-vpc"
func vpcMatches(storedVpc, caiNetworkURL string) bool {
	return storedVpc == caiNetworkURL || strings.HasSuffix(caiNetworkURL, "/"+storedVpc)
}
