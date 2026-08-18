package doctor

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bidshard/ad-event-processor/internal/config"
	"github.com/bidshard/ad-event-processor/internal/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSlotMapProbe_Name(t *testing.T) {
	assert.Equal(t, "slotmap", SlotMapProbe{}.Name())
}

func TestRunOnlySlotmapFilter(t *testing.T) {
	probes := []Probe{
		stubProbe{name: "redis", result: Result{Name: "redis", Status: StatusPass}},
		SlotMapProbe{Deps: ProbeDeps{Config: &config.Config{}}},
	}
	rep := Run(context.Background(), Options{
		Only:   []string{"slotmap"},
		Probes: probes,
	})
	require.Len(t, rep.Results, 1)
	assert.Equal(t, "slotmap", rep.Results[0].Name)
}

func TestSlotMapProbe_passWhenPostgresMatchesHTTP(t *testing.T) {
	slots := make([]uint16, domain.SlotCount)
	for i := range slots {
		slots[i] = uint16(i % 4)
	}
	payload := domain.OpsSlotMapResponse{
		Version:       1,
		ActiveVersion: 1,
		Slots:         slots,
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewEncoder(w).Encode(payload))
	}))
	t.Cleanup(srv.Close)

	probe := SlotMapProbe{
		Deps: ProbeDeps{
			Config: &config.Config{
				ManagementURL: srv.URL,
				RedisAddrs:    []string{"h0:6379", "h1:6379", "h2:6379", "h3:6379"},
			},
			SlotMapFromPG: func(context.Context) (domain.OpsSlotMapResponse, error) {
				return payload, nil
			},
		},
	}

	result := probe.Run(context.Background())
	assert.Equal(t, StatusPass, result.Status)
	assert.Contains(t, result.Detail, "v1")
}

func TestSlotMapProbe_failOnTableDrift(t *testing.T) {
	pgSlots := make([]uint16, domain.SlotCount)
	httpSlots := make([]uint16, domain.SlotCount)
	for i := range pgSlots {
		pgSlots[i] = uint16(i % 4)
		httpSlots[i] = uint16(i % 4)
	}
	httpSlots[10] = 99

	payload := domain.OpsSlotMapResponse{Version: 1, ActiveVersion: 1, Slots: httpSlots}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewEncoder(w).Encode(payload))
	}))
	t.Cleanup(srv.Close)

	probe := SlotMapProbe{
		Deps: ProbeDeps{
			Config: &config.Config{
				ManagementURL: srv.URL,
				RedisAddrs:    []string{"h0:6379", "h1:6379", "h2:6379", "h3:6379"},
			},
			SlotMapFromPG: func(context.Context) (domain.OpsSlotMapResponse, error) {
				return domain.OpsSlotMapResponse{Version: 1, ActiveVersion: 1, Slots: pgSlots}, nil
			},
		},
	}

	result := probe.Run(context.Background())
	assert.Equal(t, StatusFail, result.Status)
	assert.Contains(t, result.Detail, "slot table diffs")
}

func TestCheckHint_slotmap(t *testing.T) {
	hint := CheckHint("slotmap")
	require.Contains(t, hint, "ARCHITECTURE.md")
}
