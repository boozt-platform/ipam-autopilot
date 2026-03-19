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

package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVpcMatches(t *testing.T) {
	fullURL := "https://www.googleapis.com/compute/v1/projects/my-project/global/networks/my-vpc"

	tests := []struct {
		name      string
		storedVpc string
		caiURL    string
		want      bool
	}{
		{"full URL exact match", fullURL, fullURL, true},
		{"short network name", "my-vpc", fullURL, true},
		{"projects/... path", "projects/my-project/global/networks/my-vpc", fullURL, true},
		{"wrong name", "other-vpc", fullURL, false},
		{"partial prefix should not match", "vpc", fullURL, false},
		{"empty stored", "", fullURL, false},
		{"both empty", "", "", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, vpcMatches(tc.storedVpc, tc.caiURL))
		})
	}
}
