package domain

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const DefaultSlotMapParitySamples = 512

func LoadOpsSlotMapFromPool(ctx context.Context, pool *pgxpool.Pool) (OpsSlotMapResponse, error) {
	if pool == nil {
		return OpsSlotMapResponse{}, fmt.Errorf("slot map: nil pool")
	}
	repo := NewSlotMapRepo(pool)
	active, err := repo.GetActiveVersion(ctx)
	if err != nil {
		return OpsSlotMapResponse{}, err
	}
	meta, err := repo.GetSlotMapMeta(ctx)
	if err != nil {
		return OpsSlotMapResponse{}, err
	}
	rows, err := repo.ListVersion(ctx, active)
	if err != nil {
		return OpsSlotMapResponse{}, err
	}
	slots, err := SlotMapShardTable(rows)
	if err != nil {
		return OpsSlotMapResponse{}, err
	}
	return OpsSlotMapResponse{
		Version:       active,
		ActiveVersion: active,
		RoutingEpoch:  meta.RoutingEpoch,
		Slots:         slots,
	}, nil
}

func FetchOpsSlotMapHTTP(ctx context.Context, client *http.Client, baseURL string) (OpsSlotMapResponse, error) {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		return OpsSlotMapResponse{}, fmt.Errorf("slot map http: empty base URL")
	}
	if client == nil {
		client = &http.Client{Timeout: 3 * time.Second}
	}
	url := strings.TrimRight(baseURL, "/") + "/ops/shards/slot-map"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return OpsSlotMapResponse{}, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return OpsSlotMapResponse{}, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return OpsSlotMapResponse{}, fmt.Errorf("slot map http: status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var doc OpsSlotMapResponse
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return OpsSlotMapResponse{}, fmt.Errorf("slot map http decode: %w", err)
	}
	if len(doc.Slots) != SlotCount {
		return OpsSlotMapResponse{}, fmt.Errorf("slot map http: expected %d slots, got %d", SlotCount, len(doc.Slots))
	}
	if doc.Version == 0 {
		doc.Version = doc.ActiveVersion
	}
	return doc, nil
}

func SlotMapsEqual(a, b []uint16) bool {
	diffs, _ := CompareSlotMaps(a, b)
	return diffs == 0
}

func CompareSlotMaps(a, b []uint16) (diffs int, firstSlot int) {
	if len(a) != SlotCount || len(b) != SlotCount {
		return SlotCount, -1
	}
	firstSlot = -1
	for i := range a {
		if a[i] != b[i] {
			diffs++
			if firstSlot < 0 {
				firstSlot = i
			}
		}
	}
	return diffs, firstSlot
}

func ShardFromSlotTable(id uuid.UUID, slots []uint16) (int, bool) {
	if len(slots) != SlotCount {
		return 0, false
	}
	slot := int(crc32Castagnoli(&id) & SlotMask)
	return int(slots[slot]), true
}

func CheckSlotMapRoutingParity(sharder *StaticSlotSharder, slots []uint16, samples int) int {
	if sharder == nil || samples <= 0 || len(slots) != SlotCount {
		return samples
	}
	mismatches := 0
	for range samples {
		id := uuid.New()
		goShard := sharder.GetShard(id)
		edgeShard, ok := ShardFromSlotTable(id, slots)
		if !ok || goShard != edgeShard {
			mismatches++
		}
	}
	return mismatches
}

func ApplySlotMapToSharder(sharder *StaticSlotSharder, resp OpsSlotMapResponse) error {
	if sharder == nil {
		return fmt.Errorf("slot map: nil sharder")
	}
	if len(resp.Slots) != SlotCount {
		return ErrSlotMapIncomplete
	}
	var table [SlotCount]uint16
	copy(table[:], resp.Slots)
	sharder.StoreSlotMap(&table)
	version := resp.Version
	if version == 0 {
		version = resp.ActiveVersion
	}
	sharder.SetActiveVersion(version)
	return nil
}
