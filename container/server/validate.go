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
	"fmt"
	"log"
	"net"
	"strings"

	"github.com/gofiber/fiber/v2"
)

var errNoAddressAvailable = fmt.Errorf("no_address_range_available_in_parent")

func validateName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("name must not be empty")
	}
	if len(name) > 255 {
		return "", fmt.Errorf("name must not exceed 255 characters")
	}
	return name, nil
}

func validateCIDR(cidrStr string) error {
	ip, ipNet, err := net.ParseCIDR(cidrStr)
	if err != nil {
		return fmt.Errorf("invalid cidr %q: %v", cidrStr, err)
	}
	if ipNet.IP.To4() == nil {
		return fmt.Errorf("only IPv4 CIDRs are supported")
	}
	if !ip.Equal(ipNet.IP) {
		ones, _ := ipNet.Mask.Size()
		return fmt.Errorf("cidr has host bits set, did you mean %s/%d?", ipNet.IP, ones)
	}
	return nil
}

func validateLabels(labels map[string]string) error {
	for k, v := range labels {
		if k == "" || v == "" {
			return fmt.Errorf("label keys and values must not be empty")
		}
		if len(k) > 63 {
			return fmt.Errorf("label key %q must not exceed 63 characters", k)
		}
		if len(v) > 255 {
			return fmt.Errorf("label value for key %q must not exceed 255 characters", k)
		}
		for _, ch := range k {
			if ch < 32 || ch == 127 {
				return fmt.Errorf("label key %q contains invalid characters", k)
			}
		}
		for _, ch := range v {
			if ch < 32 || ch == 127 {
				return fmt.Errorf("label value for key %q contains invalid characters", k)
			}
		}
	}
	return nil
}

func validateRangeSize(size int) error {
	if size < 1 || size > 32 {
		return fmt.Errorf("range_size must be between 1 and 32, got %d", size)
	}
	return nil
}

func internalError(c *fiber.Ctx, err error) error {
	log.Printf("ERROR: %v", err)
	return c.Status(503).JSON(&fiber.Map{
		"success": false,
		"message": "internal server error",
	})
}
