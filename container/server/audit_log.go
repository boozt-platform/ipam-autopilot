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
	"encoding/base64"
	"encoding/json"
	"log"
	"strings"
)

const (
	ActionCreate = "create"
	ActionUpdate = "update"
	ActionDelete = "delete"

	ResourceRange  = "range"
	ResourceDomain = "domain"
)

type AuditLog struct {
	ID           int               `json:"id"`
	OccurredAt   string            `json:"occurred_at"`
	Action       string            `json:"action"`
	ResourceType string            `json:"resource_type"`
	ResourceID   int               `json:"resource_id"`
	ResourceName string            `json:"resource_name"`
	Detail       map[string]string `json:"detail,omitempty"`
}

func extractCallerFromToken(authHeader string) string {
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 {
		return ""
	}
	jwtParts := strings.Split(parts[1], ".")
	if len(jwtParts) != 3 {
		return ""
	}
	payload, err := base64.RawURLEncoding.DecodeString(jwtParts[1])
	if err != nil {
		return ""
	}
	var claims map[string]interface{}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return ""
	}
	if sub, ok := claims["sub"].(string); ok {
		return sub
	}
	return ""
}

func writeAuditLog(action, resourceType string, resourceID int, resourceName string, detail map[string]string, caller ...string) {
	if len(caller) > 0 && caller[0] != "" {
		if detail == nil {
			detail = map[string]string{}
		}
		detail["caller"] = caller[0]
	}
	var detailJSON interface{}
	if len(detail) > 0 {
		b, err := json.Marshal(detail)
		if err == nil {
			detailJSON = string(b)
		}
	}
	_, err := db.Exec(
		"INSERT INTO audit_logs (action, resource_type, resource_id, resource_name, detail) VALUES (?,?,?,?,?)",
		action, resourceType, resourceID, resourceName, detailJSON,
	)
	if err != nil {
		log.Printf("audit log write failed: %v", err)
	}
}

func GetAuditLogsFromDB(limit int) ([]AuditLog, error) {
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	rows, err := db.Query(
		"SELECT id, occurred_at, action, resource_type, resource_id, resource_name, detail FROM audit_logs ORDER BY occurred_at DESC, id DESC LIMIT ?",
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var logs []AuditLog
	for rows.Next() {
		var entry AuditLog
		var detailJSON []byte
		err := rows.Scan(&entry.ID, &entry.OccurredAt, &entry.Action, &entry.ResourceType, &entry.ResourceID, &entry.ResourceName, &detailJSON)
		if err != nil {
			return nil, err
		}
		if len(detailJSON) > 0 {
			_ = json.Unmarshal(detailJSON, &entry.Detail)
		}
		logs = append(logs, entry)
	}
	return logs, nil
}
