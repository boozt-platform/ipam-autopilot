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
	"context"
	"fmt"
	"log/slog"
	"time"
)

const caiSyncChunkSize = 200

// SyncCAISubnets fetches all Subnetwork assets from Cloud Asset Inventory for the
// given GCP organisation and upserts them into the cai_subnets table in chunks.
// Subnets not seen in this sync run are removed (stale cleanup).
// The function is idempotent and safe to call concurrently — duplicate key conflicts
// are handled via ON DUPLICATE KEY UPDATE.
func SyncCAISubnets(ctx context.Context, orgID string) error {
	slog.Info("CAI sync started", "org", orgID)
	start := time.Now()

	subnets, err := fetchCAISubnets(ctx, fmt.Sprintf("organizations/%s", orgID))
	if err != nil {
		return fmt.Errorf("CAI fetch failed: %w", err)
	}
	slog.Info("CAI fetch complete", "subnets", len(subnets))

	// Upsert in chunks — each chunk is its own transaction so a single failure
	// does not roll back the entire sync.
	for i := 0; i < len(subnets); i += caiSyncChunkSize {
		end := i + caiSyncChunkSize
		if end > len(subnets) {
			end = len(subnets)
		}
		if err := upsertCAIChunk(ctx, subnets[i:end]); err != nil {
			return fmt.Errorf("CAI upsert chunk %d–%d failed: %w", i, end, err)
		}
	}

	// Remove entries that were not touched in this sync run (subnet was deleted in GCP).
	_, err = db.ExecContext(ctx,
		"DELETE FROM cai_subnets WHERE last_synced_at < ?", start)
	if err != nil {
		return fmt.Errorf("CAI stale cleanup failed: %w", err)
	}

	slog.Info("CAI sync complete", "org", orgID, "subnets", len(subnets), "duration", time.Since(start))
	return nil
}

func upsertCAIChunk(ctx context.Context, chunk []caiSubnet) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	for _, s := range chunk {
		_, err = tx.ExecContext(ctx,
			`INSERT INTO cai_subnets (network, cidr, last_synced_at) VALUES (?, ?, NOW())
			 ON DUPLICATE KEY UPDATE last_synced_at = NOW()`,
			s.network, s.cidr)
		if err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

// StartCAISyncLoop runs an initial sync immediately (blocking) then starts a
// background goroutine that re-syncs on the given interval.
// The goroutine stops when ctx is cancelled.
func StartCAISyncLoop(ctx context.Context, orgID string, interval time.Duration) error {
	if err := SyncCAISubnets(ctx, orgID); err != nil {
		return fmt.Errorf("initial CAI sync failed: %w", err)
	}

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := SyncCAISubnets(ctx, orgID); err != nil {
					slog.Error("CAI sync failed, will retry next interval", "error", err)
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	return nil
}
