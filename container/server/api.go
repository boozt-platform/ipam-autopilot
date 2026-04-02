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
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/apparentlymart/go-cidr/cidr"
	"github.com/gofiber/fiber/v2"
)

// --- Types ---

type CreateRoutingDomainRequest struct {
	Name string   `json:"name"`
	Vpcs []string `json:"vpcs"`
}

type UpdateRoutingDomainRequest struct {
	Name JSONString      `json:"name"`
	Vpcs JSONStringArray `json:"vpcs"`
}

type RangeRequest struct {
	Parent     string            `json:"parent"`
	Name       string            `json:"name"`
	Range_size int               `json:"range_size"`
	Domain     string            `json:"domain"`
	Cidr       string            `json:"cidr"`
	Labels     map[string]string `json:"labels"`
}

type UpdateRangeRequest struct {
	Labels map[string]string `json:"labels"`
}

type RangeStats struct {
	TotalAddresses int64   `json:"total_addresses"`
	UsedAddresses  int64   `json:"used_addresses"`
	FreeAddresses  int64   `json:"free_addresses"`
	UtilizationPct float64 `json:"utilization_pct"`
}

type ImportRangeItem struct {
	Name   string            `json:"name"`
	Cidr   string            `json:"cidr"`
	Domain string            `json:"domain"`
	Parent string            `json:"parent"`
	Labels map[string]string `json:"labels"`
}

type ImportError struct {
	Name  string `json:"name"`
	Cidr  string `json:"cidr"`
	Error string `json:"error"`
}

type ImportResult struct {
	Imported int           `json:"imported"`
	Skipped  int           `json:"skipped"`
	Errors   []ImportError `json:"errors"`
}

type JSONString struct {
	Value string
	Set   bool
}

func (i *JSONString) UnmarshalJSON(data []byte) error {
	i.Set = true
	var val string
	if err := json.Unmarshal(data, &val); err != nil {
		return err
	}
	i.Value = val
	return nil
}

type JSONStringArray struct {
	Value []string
	Set   bool
}

func (i *JSONStringArray) UnmarshalJSON(data []byte) error {
	i.Set = true
	var val []string
	if err := json.Unmarshal(data, &val); err != nil {
		return err
	}
	i.Value = val
	return nil
}

// --- Functions ---

func computeStats(cidrStr string, childCIDRs []string) (RangeStats, error) {
	_, ipNet, err := net.ParseCIDR(cidrStr)
	if err != nil {
		return RangeStats{}, fmt.Errorf("invalid cidr %q: %w", cidrStr, err)
	}
	ones, bits := ipNet.Mask.Size()
	total := int64(1) << (bits - ones)

	var used int64
	for _, child := range childCIDRs {
		_, childNet, err := net.ParseCIDR(child)
		if err != nil {
			continue
		}
		childOnes, childBits := childNet.Mask.Size()
		used += int64(1) << (childBits - childOnes)
	}

	free := total - used
	var pct float64
	if total > 0 {
		pct = math.Round(float64(used)/float64(total)*10000) / 100
	}
	return RangeStats{
		TotalAddresses: total,
		UsedAddresses:  used,
		FreeAddresses:  free,
		UtilizationPct: pct,
	}, nil
}

func ImportRanges(c *fiber.Ctx) error {
	ctx := context.Background()

	var items []ImportRangeItem
	if err := c.BodyParser(&items); err != nil {
		return c.Status(400).JSON(&fiber.Map{
			"success": false,
			"message": fmt.Sprintf("Bad format %v", err),
		})
	}

	result := ImportResult{Errors: []ImportError{}}

	for _, item := range items {
		name, nameErr := validateName(item.Name)
		if nameErr != nil {
			result.Errors = append(result.Errors, ImportError{Name: item.Name, Cidr: item.Cidr, Error: nameErr.Error()})
			continue
		}
		item.Name = name
		if item.Cidr == "" {
			result.Errors = append(result.Errors, ImportError{Name: item.Name, Cidr: item.Cidr, Error: "cidr is required"})
			continue
		}
		if labelErr := validateLabels(item.Labels); labelErr != nil {
			result.Errors = append(result.Errors, ImportError{Name: item.Name, Cidr: item.Cidr, Error: labelErr.Error()})
			continue
		}

		var routingDomain *RoutingDomain
		var err error
		if item.Domain == "" {
			routingDomain, err = getDefaultRoutingDomain()
		} else {
			domainID, parseErr := strconv.ParseInt(item.Domain, 10, 64)
			if parseErr != nil {
				result.Errors = append(result.Errors, ImportError{Name: item.Name, Cidr: item.Cidr, Error: fmt.Sprintf("invalid domain: %v", parseErr)})
				continue
			}
			routingDomain, err = GetRoutingDomainFromDB(domainID)
		}
		if err != nil {
			result.Errors = append(result.Errors, ImportError{Name: item.Name, Cidr: item.Cidr, Error: fmt.Sprintf("could not resolve domain: %v", err)})
			continue
		}

		exists, err := rangeExistsByCidrAndDomain(item.Cidr, routingDomain.Id)
		if err != nil {
			result.Errors = append(result.Errors, ImportError{Name: item.Name, Cidr: item.Cidr, Error: fmt.Sprintf("database error: %v", err)})
			continue
		}
		if exists {
			result.Skipped++
			continue
		}

		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			result.Errors = append(result.Errors, ImportError{Name: item.Name, Cidr: item.Cidr, Error: fmt.Sprintf("database error: %v", err)})
			continue
		}

		parentID := int64(-1)
		if item.Parent != "" {
			parentID, err = strconv.ParseInt(item.Parent, 10, 64)
			if err != nil {
				_ = tx.Rollback()
				result.Errors = append(result.Errors, ImportError{Name: item.Name, Cidr: item.Cidr, Error: fmt.Sprintf("invalid parent: %v", err)})
				continue
			}
		}

		id, err := CreateRangeInDb(tx, parentID, routingDomain.Id, item.Name, item.Cidr, item.Labels)
		if err != nil {
			_ = tx.Rollback()
			result.Errors = append(result.Errors, ImportError{Name: item.Name, Cidr: item.Cidr, Error: fmt.Sprintf("failed to import: %v", err)})
			continue
		}
		if err = tx.Commit(); err != nil {
			result.Errors = append(result.Errors, ImportError{Name: item.Name, Cidr: item.Cidr, Error: fmt.Sprintf("commit failed: %v", err)})
			continue
		}

		caller := extractCallerFromToken(c.Get("Authorization"))
		writeAuditLog(ActionCreate, ResourceRange, int(id), item.Name, map[string]string{"cidr": item.Cidr}, caller)
		result.Imported++
	}

	return c.Status(200).JSON(result)
}

func GetRanges(c *fiber.Ctx) error {
	var results []*fiber.Map
	ranges, err := GetRangesFromDB(c.Query("name"))
	if err != nil {
		return internalError(c, err)
	}

	for i := 0; i < len(ranges); i++ {
		results = append(results, &fiber.Map{
			"id":     ranges[i].Subnet_id,
			"parent": ranges[i].Parent_id,
			"name":   ranges[i].Name,
			"cidr":   ranges[i].Cidr,
			"labels": ranges[i].Labels,
		})
	}
	return c.Status(200).JSON(results)
}

func GetRange(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(&fiber.Map{
			"success": false,
			"message": fmt.Sprintf("%v", err),
		})
	}
	rang, err := GetRangeFromDB(id)
	if err != nil {
		return internalError(c, err)
	}

	childCIDRs, err := GetChildRangeCIDRsFromDB(id)
	if err != nil {
		return internalError(c, fmt.Errorf("failed fetching child ranges: %w", err))
	}
	stats, err := computeStats(rang.Cidr, childCIDRs)
	if err != nil {
		return internalError(c, fmt.Errorf("failed computing stats: %w", err))
	}

	return c.Status(200).JSON(&fiber.Map{
		"id":     rang.Subnet_id,
		"parent": rang.Parent_id,
		"name":   rang.Name,
		"cidr":   rang.Cidr,
		"labels": rang.Labels,
		"stats":  stats,
	})
}

func UpdateRange(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(&fiber.Map{
			"success": false,
			"message": fmt.Sprintf("%v", err),
		})
	}

	p := new(UpdateRangeRequest)
	if err := c.BodyParser(p); err != nil {
		return c.Status(400).JSON(&fiber.Map{
			"success": false,
			"message": fmt.Sprintf("Bad format %v", err),
		})
	}

	if err := validateLabels(p.Labels); err != nil {
		return c.Status(400).JSON(&fiber.Map{
			"success": false,
			"message": err.Error(),
		})
	}

	rang, err := GetRangeFromDB(id)
	if err != nil {
		return c.Status(404).JSON(&fiber.Map{
			"success": false,
			"message": fmt.Sprintf("range not found: %v", err),
		})
	}

	if err := UpdateRangeLabelsInDb(id, p.Labels); err != nil {
		return internalError(c, fmt.Errorf("failed updating range: %w", err))
	}

	caller := extractCallerFromToken(c.Get("Authorization"))
	writeAuditLog(ActionUpdate, ResourceRange, int(id), rang.Name, map[string]string{"cidr": rang.Cidr}, caller)
	return c.Status(200).JSON(&fiber.Map{
		"id":     id,
		"name":   rang.Name,
		"cidr":   rang.Cidr,
		"labels": p.Labels,
	})
}

func DeleteRange(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(&fiber.Map{
			"success": false,
			"message": fmt.Sprintf("%v", err),
		})
	}
	rang, err := GetRangeFromDB(id)
	if err != nil {
		return internalError(c, err)
	}

	childCount, err := CountChildRanges(id)
	if err != nil {
		return internalError(c, err)
	}
	if childCount > 0 {
		return c.Status(409).JSON(&fiber.Map{
			"success": false,
			"message": fmt.Sprintf("range has %d child ranges; delete them first", childCount),
		})
	}

	err = DeleteRangeFromDb(id)
	if err != nil {
		return internalError(c, err)
	}

	caller := extractCallerFromToken(c.Get("Authorization"))
	writeAuditLog(ActionDelete, ResourceRange, int(id), rang.Name, map[string]string{"cidr": rang.Cidr}, caller)
	return c.Status(200).JSON(&fiber.Map{
		"success": true,
	})
}

func GetAuditLogs(c *fiber.Ctx) error {
	limit, _ := strconv.Atoi(c.Query("limit", "100"))
	logs, err := GetAuditLogsFromDB(limit)
	if err != nil {
		return internalError(c, err)
	}
	if logs == nil {
		logs = []AuditLog{}
	}
	return c.Status(200).JSON(logs)
}

func CreateNewRange(c *fiber.Ctx) error {
	ctx := context.Background()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return internalError(c, fmt.Errorf("BeginTx: %w", err))
	}

	// Instantiate new RangeRequest struct
	p := RangeRequest{}
	//  Parse body into RangeRequest struct
	if err := c.BodyParser(&p); err != nil {
		fmt.Printf("Failed parsing body. %s Bad format %v", string(c.Body()), err)
		_ = tx.Rollback()
		return c.Status(400).JSON(&fiber.Map{
			"success": false,
			"message": fmt.Sprintf("Bad format %v", err),
		})
	}

	name, nameErr := validateName(p.Name)
	if nameErr != nil {
		_ = tx.Rollback()
		return c.Status(400).JSON(&fiber.Map{
			"success": false,
			"message": nameErr.Error(),
		})
	}
	p.Name = name

	if labelErr := validateLabels(p.Labels); labelErr != nil {
		_ = tx.Rollback()
		return c.Status(400).JSON(&fiber.Map{
			"success": false,
			"message": labelErr.Error(),
		})
	}

	if p.Cidr != "" {
		if cidrErr := validateCIDR(p.Cidr); cidrErr != nil {
			_ = tx.Rollback()
			return c.Status(400).JSON(&fiber.Map{
				"success": false,
				"message": cidrErr.Error(),
			})
		}
	}

	if p.Cidr == "" {
		if sizeErr := validateRangeSize(p.Range_size); sizeErr != nil {
			_ = tx.Rollback()
			return c.Status(400).JSON(&fiber.Map{
				"success": false,
				"message": sizeErr.Error(),
			})
		}
	}

	var routingDomain *RoutingDomain
	if p.Domain == "" {
		routingDomain, err = GetDefaultRoutingDomainFromDB(tx)
		if err != nil {
			fmt.Printf("Error %v", err)
			_ = tx.Rollback()
			return c.Status(503).JSON(&fiber.Map{
				"success": false,
				"message": "Couldn't retrieve default routing domain",
			})
		}
	} else {
		domain_id, err := strconv.ParseInt(p.Domain, 10, 64)
		if err != nil {
			_ = tx.Rollback()
			return c.Status(400).JSON(&fiber.Map{
				"success": false,
				"message": fmt.Sprintf("%v", err),
			})
		}
		routingDomain, err = GetRoutingDomainFromDB(domain_id)
		if err != nil {
			fmt.Printf("Error %v", err)
			_ = tx.Rollback()
			return c.Status(503).JSON(&fiber.Map{
				"success": false,
				"message": "Couldn't retrieve default routing domain",
			})
		}
	}

	if p.Cidr != "" {
		return directInsert(c, tx, p, routingDomain)
	} else {
		return findNewLeaseAndInsert(c, tx, p, routingDomain)
	}
}

func directInsert(c *fiber.Ctx, tx *sql.Tx, p RangeRequest, routingDomain *RoutingDomain) error {
	var err error
	domain_id, err := strconv.ParseInt(p.Domain, 10, 64)
	if err != nil {
		_ = tx.Rollback()
		return c.Status(400).JSON(&fiber.Map{
			"success": false,
			"message": fmt.Sprintf("Domain needs to be an integer %v", err),
		})
	}

	parent_id := int64(-1)
	if p.Parent != "" {
		parent_id, err = strconv.ParseInt(p.Parent, 10, 64)
		if err != nil {
			rangeFromDb, err := getRangeByCidrAndRoutingDomain(tx, p.Parent, int(domain_id))
			if err != nil {
				_ = tx.Rollback()
				return c.Status(400).JSON(&fiber.Map{
					"success": false,
					"message": fmt.Sprintf("Parent needs to be either a cidr range within the routing domain or the id of a valid range %v", err),
				})
			}
			parent_id = int64(rangeFromDb.Subnet_id)
		} else {
			// When parent is an integer ID, verify it belongs to the same domain.
			parentRange, verifyErr := GetRangeFromDBWithTx(tx, parent_id)
			if verifyErr != nil {
				_ = tx.Rollback()
				return c.Status(400).JSON(&fiber.Map{
					"success": false,
					"message": fmt.Sprintf("parent range not found: %v", verifyErr),
				})
			}
			if parentRange.Routing_domain_id != int(domain_id) {
				_ = tx.Rollback()
				return c.Status(400).JSON(&fiber.Map{
					"success": false,
					"message": "parent range belongs to a different routing domain",
				})
			}
		}
	}

	id, err := CreateRangeInDb(tx, parent_id,
		int(domain_id),
		p.Name,
		p.Cidr,
		p.Labels)

	if err != nil {
		_ = tx.Rollback()
		return internalError(c, fmt.Errorf("unable to create new subnet lease: %w", err))
	}

	if err = tx.Commit(); err != nil {
		return internalError(c, err)
	}

	caller := extractCallerFromToken(c.Get("Authorization"))
	writeAuditLog(ActionCreate, ResourceRange, int(id), p.Name, map[string]string{"cidr": p.Cidr}, caller)
	return c.Status(200).JSON(&fiber.Map{
		"id":     id,
		"name":   p.Name,
		"cidr":   p.Cidr,
		"labels": p.Labels,
	})
}

func findNewLeaseAndInsert(c *fiber.Ctx, tx *sql.Tx, p RangeRequest, routingDomain *RoutingDomain) error {
	var err error
	var parent *Range
	if p.Parent != "" {
		parent_id, err := strconv.ParseInt(p.Parent, 10, 64)
		if err != nil {
			parent, err = getRangeByCidrAndRoutingDomain(tx, p.Parent, routingDomain.Id)
			if err != nil {
				_ = tx.Rollback()
				return c.Status(400).JSON(&fiber.Map{
					"success": false,
					"message": fmt.Sprintf("Parent needs to be either a cidr range within the routing domain or the id of a valid range %v", err),
				})
			}
		} else {
			parent, err = GetRangeFromDBWithTx(tx, parent_id)
			if err != nil {
				_ = tx.Rollback()
				return internalError(c, fmt.Errorf("unable to create new subnet lease: %w", err))
			}
			if parent.Routing_domain_id != routingDomain.Id {
				_ = tx.Rollback()
				return c.Status(400).JSON(&fiber.Map{
					"success": false,
					"message": "parent range belongs to a different routing domain",
				})
			}
		}
	} else {
		_ = tx.Rollback()
		return c.Status(400).JSON(&fiber.Map{
			"success": false,
			"message": "Please provide the ID of a parent range",
		})
	}
	range_size := p.Range_size
	subnet_ranges, err := GetRangesForParentFromDB(tx, int64(parent.Subnet_id))
	if err != nil {
		_ = tx.Rollback()
		return internalError(c, fmt.Errorf("unable to create new subnet lease: %w", err))
	}
	if orgID := os.Getenv("IPAM_CAI_ORG_ID"); orgID != "" && routingDomain.Vpcs != "" {
		vpcs := strings.Split(routingDomain.Vpcs, ",")
		var caiRanges []Range
		if os.Getenv("IPAM_CAI_DB_SYNC") == "TRUE" {
			caiRanges, err = GetCAISubnetsForNetworks(vpcs)
		} else {
			caiRanges, err = getLiveCAISubnetsForNetworks(context.Background(), orgID, vpcs)
		}
		if err != nil {
			_ = tx.Rollback()
			return internalError(c, fmt.Errorf("CAI lookup failed: %w", err))
		}
		for _, r := range caiRanges {
			if !ContainsRange(subnet_ranges, r.Cidr) {
				subnet_ranges = append(subnet_ranges, r)
			}
		}
	}

	subnet, subnetOnes, err := findNextSubnet(int(range_size), parent.Cidr, subnet_ranges)
	if err != nil {
		_ = tx.Rollback()
		if err.Error() == errNoAddressAvailable.Error() {
			return c.Status(409).JSON(&fiber.Map{
				"success": false,
				"message": "no address range available in parent",
			})
		}
		return internalError(c, err)
	}
	nextSubnet, _ := cidr.NextSubnet(subnet, int(range_size))
	log.Printf("next subnet will be starting with %s", nextSubnet.IP.String())

	id, err := CreateRangeInDb(tx, int64(parent.Subnet_id), routingDomain.Id, p.Name, fmt.Sprintf("%s/%d", subnet.IP.To4().String(), subnetOnes), p.Labels)

	if err != nil {
		_ = tx.Rollback()
		return internalError(c, fmt.Errorf("unable to create new subnet lease: %w", err))
	}

	if err = tx.Commit(); err != nil {
		return internalError(c, err)
	}

	allocatedCidr := fmt.Sprintf("%s/%d", subnet.IP.To4().String(), subnetOnes)
	caller := extractCallerFromToken(c.Get("Authorization"))
	writeAuditLog(ActionCreate, ResourceRange, int(id), p.Name, map[string]string{"cidr": allocatedCidr}, caller)
	return c.Status(200).JSON(&fiber.Map{
		"id":     id,
		"name":   p.Name,
		"cidr":   allocatedCidr,
		"labels": p.Labels,
	})
}

func findNextSubnet(range_size int, sourceRange string, existingRanges []Range) (*net.IPNet, int, error) {
	_, parentNet, err := net.ParseCIDR(sourceRange)
	if err != nil {
		return nil, -1, err
	}

	subnet, subnetOnes, err := createNewSubnetLease(sourceRange, range_size, 0)
	if err != nil {
		return nil, -1, err
	}
	log.Printf("new subnet lease %s/%d", subnet.IP.String(), subnetOnes)

	var lastSubnet = false
	for {
		err = verifyNoOverlap(sourceRange, existingRanges, subnet)
		if err == nil {
			break
		} else if !lastSubnet {
			subnet, lastSubnet = cidr.NextSubnet(subnet, int(range_size))
			if !parentNet.Contains(subnet.IP) {
				return nil, -1, fmt.Errorf("no_address_range_available_in_parent")
			}
		} else {
			return nil, -1, err
		}
	}

	return subnet, subnetOnes, nil
}

func GetRoutingDomain(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(&fiber.Map{
			"success": false,
			"message": fmt.Sprintf("%v", err),
		})
	}
	domain, err := GetRoutingDomainFromDB(id)
	if err != nil {
		return internalError(c, err)
	}

	return c.Status(200).JSON(&fiber.Map{
		"id":   domain.Id,
		"name": domain.Name,
		"vpcs": domain.Vpcs,
	})
}

func DeleteRoutingDomain(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(&fiber.Map{
			"success": false,
			"message": fmt.Sprintf("%v", err),
		})
	}
	domain, err := GetRoutingDomainFromDB(id)
	if err != nil {
		return internalError(c, err)
	}

	rangeCount, err := CountRangesForDomain(id)
	if err != nil {
		return internalError(c, err)
	}
	if rangeCount > 0 {
		return c.Status(409).JSON(&fiber.Map{
			"success": false,
			"message": fmt.Sprintf("routing domain has %d ranges; delete them first", rangeCount),
		})
	}

	err = DeleteRoutingDomainFromDB(id)
	if err != nil {
		return internalError(c, err)
	}

	caller := extractCallerFromToken(c.Get("Authorization"))
	writeAuditLog(ActionDelete, ResourceDomain, int(id), domain.Name, nil, caller)
	return c.Status(200).JSON(&fiber.Map{})
}

func GetRoutingDomains(c *fiber.Ctx) error {
	var results []*fiber.Map
	domains, err := GetRoutingDomainsFromDB()
	if err != nil {
		return internalError(c, err)
	}

	for i := 0; i < len(domains); i++ {
		results = append(results, &fiber.Map{
			"id":   domains[i].Id,
			"name": domains[i].Name,
			"vpcs": domains[i].Vpcs,
		})
	}

	return c.Status(200).JSON(results)
}

func UpdateRoutingDomain(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(&fiber.Map{
			"success": false,
			"message": fmt.Sprintf("%v", err),
		})
	}

	// Instantiate new UpdateRoutingDomainRequest struct
	p := new(UpdateRoutingDomainRequest)
	//  Parse body into UpdateRoutingDomainRequest struct
	if err := c.BodyParser(p); err != nil {
		return c.Status(400).JSON(&fiber.Map{
			"success": false,
			"message": fmt.Sprintf("Bad format %v", err),
		})
	}

	if p.Name.Set {
		name, nameErr := validateName(p.Name.Value)
		if nameErr != nil {
			return c.Status(400).JSON(&fiber.Map{
				"success": false,
				"message": nameErr.Error(),
			})
		}
		p.Name.Value = name
	}

	err = UpdateRoutingDomainOnDb(id, p.Name, p.Vpcs)
	if err != nil {
		return internalError(c, fmt.Errorf("unable to update routing domain: %w", err))
	}
	return c.Status(200).JSON(&fiber.Map{})
}

func CreateRoutingDomain(c *fiber.Ctx) error {
	// Instantiate new CreateRoutingDomainRequest struct
	p := new(CreateRoutingDomainRequest)
	//  Parse body into CreateRoutingDomainRequest struct
	if err := c.BodyParser(p); err != nil {
		return c.Status(400).JSON(&fiber.Map{
			"success": false,
			"message": fmt.Sprintf("Bad format %v", err),
		})
	}

	name, nameErr := validateName(p.Name)
	if nameErr != nil {
		return c.Status(400).JSON(&fiber.Map{
			"success": false,
			"message": nameErr.Error(),
		})
	}
	p.Name = name

	id, err := CreateRoutingDomainOnDb(p.Name, p.Vpcs)
	if err != nil {
		return internalError(c, fmt.Errorf("unable to create new routing domain: %w", err))
	}

	caller := extractCallerFromToken(c.Get("Authorization"))
	writeAuditLog(ActionCreate, ResourceDomain, int(id), p.Name, nil, caller)
	return c.Status(200).JSON(&fiber.Map{
		"id": id,
	})
}

func ContainsRange(array []Range, cidr string) bool {
	for i := 0; i < len(array); i++ {
		if cidr == array[i].Cidr {
			return true
		}
	}
	return false
}
