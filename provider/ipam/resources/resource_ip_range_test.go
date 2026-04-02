// Copyright 2026 Boozt Fashion AB
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0

package resources

import (
	"testing"
)

func TestResolveRangeSize(t *testing.T) {
	tests := []struct {
		name      string
		rangeSize int
		cidr      string
		want      int
		wantErr   bool
	}{
		{
			name:      "derives from /8 cidr",
			rangeSize: 0,
			cidr:      "10.0.0.0/8",
			want:      8,
		},
		{
			name:      "derives from /22 cidr",
			rangeSize: 0,
			cidr:      "10.0.4.0/22",
			want:      22,
		},
		{
			name:      "derives from /28 cidr",
			rangeSize: 0,
			cidr:      "192.168.1.0/28",
			want:      28,
		},
		{
			name:      "explicit range_size used when cidr not set",
			rangeSize: 24,
			cidr:      "",
			want:      24,
		},
		{
			name:      "explicit range_size takes precedence when both set",
			rangeSize: 24,
			cidr:      "10.0.0.0/24",
			want:      24,
		},
		{
			name:    "error when both empty",
			wantErr: true,
		},
		{
			name:    "error on invalid cidr",
			cidr:    "not-a-cidr",
			wantErr: true,
		},
		{
			name:      "plain ip without prefix - range_size provided explicitly",
			rangeSize: 24,
			cidr:      "10.0.0.0",
			want:      24,
		},
		{
			name:      "plain ip without prefix - no range_size - error",
			rangeSize: 0,
			cidr:      "10.0.0.0",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveRangeSize(tt.rangeSize, tt.cidr)
			if (err != nil) != tt.wantErr {
				t.Errorf("resolveRangeSize() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("resolveRangeSize() = %d, want %d", got, tt.want)
			}
		})
	}
}
