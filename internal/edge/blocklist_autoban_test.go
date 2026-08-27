package edge

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type autoBanStub struct {
	sets  map[string][]string
	zsets map[string]map[string]float64
}

func (s *autoBanStub) SMembers(_ context.Context, key string) *redis.StringSliceCmd {
	cmd := redis.NewStringSliceCmd(context.Background())
	cmd.SetVal(append([]string(nil), s.sets[key]...))
	return cmd
}

func (s *autoBanStub) ZScore(_ context.Context, key, member string) *redis.FloatCmd {
	cmd := redis.NewFloatCmd(context.Background())
	if s.zsets[key] == nil {
		cmd.SetErr(redis.Nil)
		return cmd
	}
	score, ok := s.zsets[key][member]
	if !ok {
		cmd.SetErr(redis.Nil)
		return cmd
	}
	cmd.SetVal(score)
	return cmd
}

func (s *autoBanStub) SRem(_ context.Context, key string, members ...interface{}) *redis.IntCmd {
	cmd := redis.NewIntCmd(context.Background())
	for _, m := range members {
		ip := m.(string)
		next := s.sets[key][:0]
		for _, v := range s.sets[key] {
			if v != ip {
				next = append(next, v)
			}
		}
		s.sets[key] = next
	}
	cmd.SetVal(1)
	return cmd
}

func (s *autoBanStub) ZRem(_ context.Context, key string, members ...interface{}) *redis.IntCmd {
	cmd := redis.NewIntCmd(context.Background())
	for _, m := range members {
		delete(s.zsets[key], m.(string))
	}
	cmd.SetVal(1)
	return cmd
}

func (s *autoBanStub) ZRangeByScore(_ context.Context, key string, opt *redis.ZRangeBy) *redis.StringSliceCmd {
	cmd := redis.NewStringSliceCmd(context.Background())
	z := s.zsets[key]
	if z == nil {
		cmd.SetVal(nil)
		return cmd
	}
	out := make([]string, 0, len(z))
	minScore, exclusive := parseZMin(opt)
	for member, score := range z {
		if opt != nil && opt.Min != "" {
			if exclusive {
				if score <= minScore {
					continue
				}
			} else if score < minScore {
				continue
			}
		}
		out = append(out, member)
	}
	cmd.SetVal(out)
	return cmd
}

func (s *autoBanStub) ZRangeByScoreWithScores(_ context.Context, key string, opt *redis.ZRangeBy) *redis.ZSliceCmd {
	cmd := redis.NewZSliceCmd(context.Background())
	z := s.zsets[key]
	if z == nil {
		cmd.SetVal(nil)
		return cmd
	}
	minScore, exclusive := parseZMin(opt)
	out := make([]redis.Z, 0, len(z))
	for member, score := range z {
		if opt != nil && opt.Min != "" {
			if exclusive {
				if score <= minScore {
					continue
				}
			} else if score < minScore {
				continue
			}
		}
		out = append(out, redis.Z{Score: score, Member: member})
	}
	cmd.SetVal(out)
	return cmd
}

func parseZMin(opt *redis.ZRangeBy) (float64, bool) {
	if opt == nil || opt.Min == "" || opt.Min == "-inf" {
		return 0, false
	}
	if len(opt.Min) > 0 && opt.Min[0] == '(' {
		var f float64
		_, _ = fmt.Sscanf(opt.Min[1:], "%f", &f)
		return f, true
	}
	var f float64
	_, _ = fmt.Sscanf(opt.Min, "%f", &f)
	return f, false
}

func TestActiveAutoBans_expiredLeaseRemoved(t *testing.T) {
	stub := &autoBanStub{
		sets: map[string][]string{
			redisKeyBlacklistAuto: {"203.0.113.55"},
		},
		zsets: map[string]map[string]float64{
			redisKeyBlacklistAutoTTL: {
				"203.0.113.55": float64(time.Now().Add(-time.Minute).Unix()),
			},
		},
	}
	active, err := activeAutoBans(context.Background(), stub)
	require.NoError(t, err)
	assert.Empty(t, active)
	assert.Empty(t, stub.sets[redisKeyBlacklistAuto])
}
