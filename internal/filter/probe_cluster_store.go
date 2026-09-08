package filter

import (
	"context"
	_ "embed"
	"fmt"
	"strconv"
	"time"

	"ad-event-processor/pkg/probecluster"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const probeClusterKeyPrefix = "probe:cluster:"

//go:embed scripts/probe_cluster_observe.lua
var probeClusterObserveLua string

var probeClusterObserveScript = redis.NewScript(probeClusterObserveLua)

type ProbeClusterObserveInput struct {
	ClusterID  [16]byte
	SessionID  string
	CampaignID uuid.UUID
	ProbeScore uint8
	VerifyIncr bool
	JA3        string
	JA4        string
	TCPSig     uint32
	TCPSigSet  uint8
	WebGLHex   string
}

type ProbeClusterStore struct {
	client redis.UniversalClient
	ttl    time.Duration
}

func NewProbeClusterStore(client redis.UniversalClient, ttl time.Duration) *ProbeClusterStore {
	if client == nil || ttl <= 0 {
		return nil
	}
	return &ProbeClusterStore{client: client, ttl: ttl}
}

func (st *ProbeClusterStore) Observe(ctx context.Context, in ProbeClusterObserveInput) (probecluster.State, error) {
	if st == nil || st.client == nil {
		return probecluster.State{}, nil
	}
	if in.SessionID == "" || in.CampaignID == uuid.Nil {
		return probecluster.State{}, nil
	}
	hexID := probecluster.ClusterIDHex(in.ClusterID)
	metaKey := probeClusterKeyPrefix + hexID + ":meta"
	sessKey := probeClusterKeyPrefix + hexID + ":sessions"
	campKey := probeClusterKeyPrefix + hexID + ":campaigns"
	tcpSig := ""
	if in.TCPSigSet != 0 {
		tcpSig = strconv.FormatUint(uint64(in.TCPSig), 16)
	}
	verify := 0
	if in.VerifyIncr {
		verify = 1
	}
	res, err := probeClusterObserveScript.Run(ctx, st.client, []string{metaKey, sessKey, campKey},
		in.SessionID,
		in.CampaignID.String(),
		int(in.ProbeScore),
		verify,
		int(st.ttl.Seconds()),
		in.JA3,
		in.JA4,
		tcpSig,
		in.WebGLHex,
	).Int64Slice()
	if err != nil {
		return probecluster.State{}, err
	}
	if len(res) < 4 {
		return probecluster.State{}, fmt.Errorf("probe cluster observe: short reply %d", len(res))
	}
	return probecluster.State{
		SessionCount:  res[0],
		CampaignCount: res[1],
		VerifyCount:   res[2],
		AvgProbeScore: uint8(res[3]),
	}, nil
}

type ProbeClusterSummary struct {
	ClusterID     string
	SessionCount  int64
	CampaignCount int64
	VerifyCount   int64
	AvgProbeScore uint8
	JA3           string
	JA4           string
	TCPSig        string
	WebGL         string
	ExportDone    bool
}

func (st *ProbeClusterStore) GetSummary(ctx context.Context, clusterIDHex string) (ProbeClusterSummary, bool, error) {
	if st == nil || st.client == nil || clusterIDHex == "" {
		return ProbeClusterSummary{}, false, nil
	}
	metaKey := probeClusterKeyPrefix + clusterIDHex + ":meta"
	fields, err := st.client.HGetAll(ctx, metaKey).Result()
	if err != nil {
		return ProbeClusterSummary{}, false, err
	}
	if len(fields) == 0 {
		return ProbeClusterSummary{}, false, nil
	}
	out := ProbeClusterSummary{
		ClusterID:  clusterIDHex,
		JA3:        fields["ja3"],
		JA4:        fields["ja4"],
		TCPSig:     fields["tcp_sig"],
		WebGL:      fields["webgl"],
		ExportDone: fields["export_done"] == "1",
	}
	out.SessionCount = atoi64(fields["session_count"])
	out.CampaignCount = atoi64(fields["campaign_count"])
	out.VerifyCount = atoi64(fields["verify_count"])
	sum := atoi64(fields["probe_score_sum"])
	n := atoi64(fields["probe_score_n"])
	if n > 0 {
		out.AvgProbeScore = uint8(sum / n)
	}
	return out, true, nil
}

func atoi64(s string) int64 {
	if s == "" {
		return 0
	}
	v, _ := strconv.ParseInt(s, 10, 64)
	return v
}

func (st *ProbeClusterStore) MarkExported(ctx context.Context, clusterIDHex string) error {
	if st == nil || st.client == nil || clusterIDHex == "" {
		return nil
	}
	metaKey := probeClusterKeyPrefix + clusterIDHex + ":meta"
	return st.client.HSet(ctx, metaKey, "export_done", "1").Err()
}

func (st *ProbeClusterStore) ScanMetaKeys(ctx context.Context, cursor uint64, count int64) ([]string, uint64, error) {
	if st == nil || st.client == nil {
		return nil, 0, nil
	}
	return st.client.Scan(ctx, cursor, probeClusterKeyPrefix+"*:meta", count).Result()
}
