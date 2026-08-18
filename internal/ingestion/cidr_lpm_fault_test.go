package ingestion

import (
	"context"
	"math/rand"
	"net/netip"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/bidshard/ad-event-processor/internal/config"
)

func TestCIDR_RCUSwap_ConcurrentReaders(t *testing.T) {
	tableA, prefsA := buildTestTable(t, "10.0.0.0/8", "54.0.0.0/8")
	tableB, prefsB := buildTestTable(t, "10.0.0.0/8", "54.0.0.0/8", "185.220.0.0/16")

	table := NewCIDRTable()
	table.Publish(mustSnapshotGen(t, tableA, 1))

	stop := make(chan struct{})
	var wg sync.WaitGroup
	for w := 0; w < 8; w++ {
		wg.Add(1)
		go func(seed int64) {
			defer wg.Done()
			rng := rand.New(rand.NewSource(seed))
			for {
				select {
				case <-stop:
					return
				default:
				}
				var a4 [4]byte
				rng.Read(a4[:])
				got, _ := table.Match4(a4)
				ip := netip.AddrFrom4(a4)
				okA := oracleMatch(prefsA, ip.String())
				okB := oracleMatch(prefsB, ip.String())
				if got != okA && got != okB {
					t.Errorf("torn read: Match4(%v)=%v, gens allow %v or %v", ip, got, okA, okB)
					return
				}
			}
		}(int64(w))
	}

	for i := 0; i < 1000; i++ {
		if i%2 == 0 {
			table.Publish(mustSnapshotGen(t, tableB, 2))
		} else {
			table.Publish(mustSnapshotGen(t, tableA, 1))
		}
	}
	close(stop)
	wg.Wait()
	t.Log("fault_proof fault=rcu_swap_1000x harness=cidr_lpm_rcu readers=8")
}

func mustSnapshotGen(t *testing.T, src *CIDRTable, gen uint64) *cidrSnapshot {
	t.Helper()
	snap := src.active.Load()
	if snap == nil {
		t.Fatal("source table not initialized")
	}
	return &cidrSnapshot{gen: gen, root4: snap.root4, root6: snap.root6, nodes: snap.nodes, prefs: snap.prefs}
}

func TestCIDR_FeedRefreshFailClosed_RetainsSnapshot(t *testing.T) {
	dir := t.TempDir()
	writeFeedFile(t, dir, "aws.json", `{"prefixes":[{"ip_prefix":"54.0.0.0/8"}]}`)

	cfg := &config.Config{
		CIDRL1Enabled:          true,
		CIDRFeedDir:            dir,
		CIDRFeedRefresh:        time.Hour,
		CIDRFeedDownloadEnable: false,
	}
	table := NewCIDRTable()
	loader := NewCIDRFeedLoader(cfg, table)
	if loader == nil {
		t.Fatal("loader nil with CIDR_L1_ENABLED=true")
	}
	loader.refreshOnce(context.Background())
	if !table.Ready() {
		t.Fatal("expected published snapshot after initial load")
	}
	if ok, feed := table.MatchIP("54.1.2.3"); !ok || feed != CIDRFeedAWS {
		t.Fatalf("expected aws match for 54.1.2.3, got ok=%v feed=%d", ok, feed)
	}

	writeFeedFile(t, dir, "aws.json", `{not json`)
	loader.refreshOnce(context.Background())
	if !table.Ready() {
		t.Fatal("snapshot lost after feed corruption")
	}
	if ok, _ := table.MatchIP("54.1.2.3"); !ok {
		t.Fatal("previous snapshot not retained after refresh failure")
	}
	t.Log("fault_proof fault=feed_corrupt_retain_snapshot harness=cidr_lpm_rcu")
}

func TestCIDR_FeedRefreshFailClosed_FirstBootFailOpen(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{
		CIDRL1Enabled:   true,
		CIDRFeedDir:     dir,
		CIDRFeedRefresh: time.Hour,
	}
	table := NewCIDRTable()
	loader := NewCIDRFeedLoader(cfg, table)
	loader.refreshOnce(context.Background())
	if table.Ready() {
		t.Fatal("table published with zero feed data")
	}
	if ok, _ := table.MatchIP("54.1.2.3"); ok {
		t.Fatal("match on uninitialized table")
	}
	t.Log("fault_proof fault=first_boot_no_feeds_fail_open harness=cidr_lpm_rcu")
}

func TestCIDR_FeedLoader_LineFormats(t *testing.T) {
	dir := t.TempDir()
	writeFeedFile(t, dir, "tor.txt", "# exit nodes\n185.220.101.1\n185.220.101.2  # inline\njunk line\n2001:db8::1\n")
	writeFeedFile(t, dir, "other.txt", "203.0.113.0/24\n")

	cfg := &config.Config{CIDRL1Enabled: true, CIDRFeedDir: dir, CIDRFeedRefresh: time.Hour}
	table := NewCIDRTable()
	NewCIDRFeedLoader(cfg, table).refreshOnce(context.Background())

	cases := []struct {
		ip   string
		want bool
		feed uint8
	}{
		{"185.220.101.1", true, CIDRFeedTor},
		{"185.220.101.2", true, CIDRFeedTor},
		{"2001:db8::1", true, CIDRFeedTor},
		{"203.0.113.55", true, CIDRFeedOther},
		{"203.0.114.1", false, 0},
		{"8.8.8.8", false, 0},
	}
	for _, tc := range cases {
		got, feed := table.MatchIP(tc.ip)
		if got != tc.want || (tc.want && feed != tc.feed) {
			t.Fatalf("MatchIP(%q) = (%v,%d), want (%v,%d)", tc.ip, got, feed, tc.want, tc.feed)
		}
	}
}

func TestCIDR_FeedLoader_AzureJSON(t *testing.T) {
	dir := t.TempDir()
	writeFeedFile(t, dir, "azure.json", `{"values":[{"properties":{"addressPrefixes":["20.190.128.0/18","2603:1000::/24"]}}]}`)

	cfg := &config.Config{CIDRL1Enabled: true, CIDRFeedDir: dir, CIDRFeedRefresh: time.Hour}
	table := NewCIDRTable()
	NewCIDRFeedLoader(cfg, table).refreshOnce(context.Background())

	if ok, feed := table.MatchIP("20.190.160.1"); !ok || feed != CIDRFeedAzure {
		t.Fatalf("azure v4 match failed: ok=%v feed=%d", ok, feed)
	}
	if ok, feed := table.MatchIP("2603:1000::1"); !ok || feed != CIDRFeedAzure {
		t.Fatalf("azure v6 match failed: ok=%v feed=%d", ok, feed)
	}
	if ok, _ := table.MatchIP("20.191.0.1"); ok {
		t.Fatal("out-of-range azure IP matched")
	}
}

func writeFeedFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCIDR_FeedLoader_DisabledWhenConfigOff(t *testing.T) {
	if l := NewCIDRFeedLoader(nil, NewCIDRTable()); l != nil {
		t.Fatal("nil config must disable loader")
	}
	if l := NewCIDRFeedLoader(&config.Config{CIDRL1Enabled: false}, NewCIDRTable()); l != nil {
		t.Fatal("CIDR_L1_ENABLED=false must disable loader")
	}
	if l := NewCIDRFeedLoader(&config.Config{CIDRL1Enabled: true}, nil); l != nil {
		t.Fatal("nil table must disable loader")
	}
}
