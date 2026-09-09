package filter

import (
	"context"
	"encoding/hex"
	"strconv"
	"strings"
	"time"

	"ad-event-processor/pkg/crowdwave"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const crowdWaveKeyPrefix = "probe:wave:"

type CrowdWaveStore struct {
	client redis.UniversalClient
	window time.Duration
	policy crowdwave.WavePolicy
}

func NewCrowdWaveStore(client redis.UniversalClient, window time.Duration, policy crowdwave.WavePolicy) *CrowdWaveStore {
	if client == nil || window <= 0 {
		return nil
	}
	return &CrowdWaveStore{client: client, window: window, policy: policy}
}

func (st *CrowdWaveStore) Observe(ctx context.Context, campaignID uuid.UUID, clusterID [16]byte, simhash uint64, nowUnix int64) (crowdwave.WaveState, error) {
	if st == nil || st.client == nil || campaignID == uuid.Nil || clusterID[0] == 0 {
		return crowdwave.WaveState{}, nil
	}
	key := crowdWaveKey(campaignID)
	entry := formatWaveEntry(clusterID, simhash, nowUnix)
	pipe := st.client.Pipeline()
	pipe.LPush(ctx, key, entry)
	pipe.LTrim(ctx, key, 0, crowdwave.DefaultWaveRingSize-1)
	pipe.Expire(ctx, key, st.window)
	if _, err := pipe.Exec(ctx); err != nil {
		return crowdwave.WaveState{}, err
	}
	raw, err := st.client.LRange(ctx, key, 0, -1).Result()
	if err != nil {
		return crowdwave.WaveState{}, err
	}
	return evaluateWaveEntries(raw, nowUnix, st.policy), nil
}

func (st *CrowdWaveStore) Snapshot(ctx context.Context, campaignID uuid.UUID) (crowdwave.WaveState, error) {
	if st == nil || st.client == nil || campaignID == uuid.Nil {
		return crowdwave.WaveState{}, nil
	}
	states, err := st.SnapshotBatch(ctx, []uuid.UUID{campaignID})
	if err != nil {
		return crowdwave.WaveState{}, err
	}
	return states[campaignID], nil
}

func (st *CrowdWaveStore) SnapshotBatch(ctx context.Context, campaignIDs []uuid.UUID) (map[uuid.UUID]crowdwave.WaveState, error) {
	if st == nil || st.client == nil || len(campaignIDs) == 0 {
		return map[uuid.UUID]crowdwave.WaveState{}, nil
	}
	nowUnix := time.Now().Unix()
	pipe := st.client.Pipeline()
	cmds := make([]*redis.StringSliceCmd, 0, len(campaignIDs))
	idOrder := make([]uuid.UUID, 0, len(campaignIDs))
	for _, campaignID := range campaignIDs {
		if campaignID == uuid.Nil {
			continue
		}
		idOrder = append(idOrder, campaignID)
		cmds = append(cmds, pipe.LRange(ctx, crowdWaveKey(campaignID), 0, -1))
	}
	if len(cmds) == 0 {
		return map[uuid.UUID]crowdwave.WaveState{}, nil
	}
	if _, err := pipe.Exec(ctx); err != nil {
		return nil, err
	}
	out := make(map[uuid.UUID]crowdwave.WaveState, len(idOrder))
	for i, campaignID := range idOrder {
		raw, err := cmds[i].Result()
		if err != nil {
			return nil, err
		}
		out[campaignID] = evaluateWaveEntries(raw, nowUnix, st.policy)
	}
	return out, nil
}

func evaluateWaveEntries(raw []string, nowUnix int64, policy crowdwave.WavePolicy) crowdwave.WaveState {
	entries := parseWaveEntries(raw)
	return crowdwave.EvaluateWave(entries, nowUnix, policy)
}

func (st *CrowdWaveStore) ListCampaignIDs(ctx context.Context) ([]uuid.UUID, error) {
	if st == nil || st.client == nil {
		return nil, nil
	}
	var out []uuid.UUID
	var cursor uint64
	for {
		keys, next, err := st.client.Scan(ctx, cursor, crowdWaveKeyPrefix+"*", 64).Result()
		if err != nil {
			return out, err
		}
		for _, key := range keys {
			idStr := strings.TrimPrefix(key, crowdWaveKeyPrefix)
			id, err := uuid.Parse(idStr)
			if err != nil || id == uuid.Nil {
				continue
			}
			out = append(out, id)
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}
	return out, nil
}

func crowdWaveKey(campaignID uuid.UUID) string {
	return crowdWaveKeyPrefix + campaignID.String()
}

func formatWaveEntry(clusterID [16]byte, simhash uint64, nowUnix int64) string {
	return hex.EncodeToString(clusterID[:]) + ":" + strconv.FormatUint(simhash, 16) + ":" + strconv.FormatInt(nowUnix, 10)
}

func parseWaveEntries(raw []string) []crowdwave.WaveEntry {
	out := make([]crowdwave.WaveEntry, 0, len(raw))
	for _, line := range raw {
		e, ok := parseWaveEntry(line)
		if ok {
			out = append(out, e)
		}
	}
	return out
}

func parseWaveEntry(line string) (crowdwave.WaveEntry, bool) {
	parts := strings.Split(line, ":")
	if len(parts) != 3 {
		return crowdwave.WaveEntry{}, false
	}
	b, err := hex.DecodeString(parts[0])
	if err != nil || len(b) != 16 {
		return crowdwave.WaveEntry{}, false
	}
	simhash, err := strconv.ParseUint(parts[1], 16, 64)
	if err != nil {
		return crowdwave.WaveEntry{}, false
	}
	observed, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return crowdwave.WaveEntry{}, false
	}
	var id [16]byte
	copy(id[:], b)
	return crowdwave.WaveEntry{ClusterID: id, Simhash: simhash, Observed: observed}, true
}
