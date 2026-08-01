package ingestion

import (
	"context"
	"os/exec"
	"strings"
	"testing"
	"time"

	"espx/internal/database"
	"espx/internal/domain"
	"espx/pkg/piihash"

	"github.com/google/uuid"
	redis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

type segmentTestRegistry struct {
	camps map[uuid.UUID]*domain.Campaign
}

func (r *segmentTestRegistry) Exists(uuid.UUID) bool { return true }
func (r *segmentTestRegistry) Add(uuid.UUID, uuid.UUID, *uuid.UUID, string, domain.PacingMode, int64, string, int32, int32, []string) {
}
func (r *segmentTestRegistry) GetCustomerID(uuid.UUID) (uuid.UUID, bool) { return uuid.Nil, false }
func (r *segmentTestRegistry) GetCampaign(id uuid.UUID) (*domain.Campaign, bool) {
	if r.camps == nil {
		return nil, false
	}
	c, ok := r.camps[id]
	return c, ok
}
func (r *segmentTestRegistry) Sync(context.Context) (int, error)        { return 0, nil }
func (r *segmentTestRegistry) StartSync(context.Context, time.Duration) {}
func (r *segmentTestRegistry) Wait(context.Context) error               { return nil }

type segmentTestRepo struct {
	camp *domain.Campaign
}

func (r *segmentTestRepo) GetByID(context.Context, uuid.UUID) (*domain.Campaign, error) {
	return r.camp, nil
}

func TestSegmentIntegration_conversionExcludeAndTTL(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	rdb, cleanup := database.SetupTestRedis(t)
	defer cleanup()

	hasher := piihash.TestHasher()
	rdbs := []redis.UniversalClient{rdb}
	segmentID := uuid.New()
	userID := "retarget-user-1"
	userHash := hasher.HashUserID(userID)

	srcCampID := uuid.New()
	dstCampID := uuid.New()
	repo := &segmentTestRepo{
		camp: &domain.Campaign{
			ID:                srcCampID,
			RetargetSegmentID: segmentID,
			SegmentTTLHours:   1,
		},
	}
	handler := NewSegmentConversionHandler(repo, nil, rdbs, hasher)
	handler.Handle(&domain.Event{
		CampaignID: srcCampID,
		Type:       conversionEventType,
		UserID:     userID,
	}, "1-0")

	member, err := segmentMemberExists(ctx, rdbs, segmentID, userHash)
	require.NoError(t, err)
	require.True(t, member)

	reg := &segmentTestRegistry{
		camps: map[uuid.UUID]*domain.Campaign{
			dstCampID: {
				ID:               dstCampID,
				SegmentExcludeID: segmentID,
			},
		},
	}
	filter := NewSegmentFilter(rdbs, reg, hasher)
	evt := &domain.Event{CampaignID: dstCampID, UserID: userID}
	require.ErrorIs(t, filter.Check(ctx, evt), ErrSegmentExcluded)

	require.NoError(t, addSegmentMember(ctx, rdbs, segmentID, userHash, time.Second))
	time.Sleep(1100 * time.Millisecond)
	member, err = segmentMemberExists(ctx, rdbs, segmentID, userHash)
	require.NoError(t, err)
	require.False(t, member)
	require.NoError(t, filter.Check(ctx, evt))
}

type segmentGetMock struct {
	mockRedisClient
	hit bool
}

func (m *segmentGetMock) Get(ctx context.Context, key string) *redis.StringCmd {
	if m.hit {
		staticStringCmd.SetVal("1")
		staticStringCmd.SetErr(nil)
	} else {
		staticStringCmd.SetVal("")
		staticStringCmd.SetErr(redis.Nil)
	}
	return staticStringCmd
}

func setupSegmentFilterBench(t testing.TB, member bool) (*SegmentFilter, *domain.Event, context.Context) {
	t.Helper()
	segmentID := uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8")
	campID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	rdbs := []redis.UniversalClient{&segmentGetMock{hit: member}}
	reg := &segmentTestRegistry{
		camps: map[uuid.UUID]*domain.Campaign{
			campID: {
				ID:               campID,
				SegmentExcludeID: segmentID,
			},
		},
	}
	f := NewSegmentFilter(rdbs, reg, piihash.TestHasher())
	evt := &domain.Event{
		CampaignID: campID,
		UserID:     "bench-user",
	}
	ctx := context.Background()
	for i := 0; i < 1000; i++ {
		_ = f.Check(ctx, evt)
	}
	return f, evt, ctx
}

func BenchmarkSegmentCheck_miss(b *testing.B) {
	f, evt, ctx := setupSegmentFilterBench(b, false)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = f.Check(ctx, evt)
	}
}

func BenchmarkSegmentCheck_hit(b *testing.B) {
	f, evt, ctx := setupSegmentFilterBench(b, true)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = f.Check(ctx, evt)
	}
}

func TestSegmentMemberExists_zeroAlloc(t *testing.T) {
	segmentID := uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8")
	rdbs := []redis.UniversalClient{&segmentGetMock{hit: false}}
	userHash := piihash.TestHasher().HashUserID("bench-user")
	ctx := context.Background()
	for i := 0; i < 1000; i++ {
		_, _ = segmentMemberExists(ctx, rdbs, segmentID, userHash)
	}
	avg := testing.AllocsPerRun(100, func() {
		_, _ = segmentMemberExists(ctx, rdbs, segmentID, userHash)
	})
	if avg > 0 {
		t.Fatalf("segmentMemberExists allocated %.1f times per run, want 0", avg)
	}
}

func TestSegmentCheck_zeroAlloc(t *testing.T) {
	f, evt, ctx := setupSegmentFilterBench(t, false)
	evt.HasUserPIIHash = true
	evt.UserPIIHash = piihash.TestHasher().HashUserID("bench-user")
	avg := testing.AllocsPerRun(100, func() {
		_ = f.Check(ctx, evt)
	})
	if avg > 0 {
		t.Fatalf("SegmentFilter.Check allocated %.1f times per run, want 0", avg)
	}
}

func TestSegmentCheck_escapeClean(t *testing.T) {
	if testing.Short() {
		t.Skip("escape analysis build")
	}
	root := moduleRootAds(t)
	cmd := exec.Command("go", "build", "-gcflags=-m", "-o", "/dev/null", "./internal/ingestion")
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("escape analysis build failed: %v\n%s", err, out)
	}
	for _, line := range strings.Split(string(out), "\n") {
		if !strings.Contains(line, "SegmentFilter).Check") {
			continue
		}
		if strings.Contains(line, "escapes to heap") {
			t.Fatalf("SegmentFilter.Check escapes to heap: %s", strings.TrimSpace(line))
		}
	}
}
