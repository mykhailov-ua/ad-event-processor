package filter

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	staticCmd       = redis.NewCmd(context.Background())
	staticStatusCmd = redis.NewStatusCmd(context.Background())
	staticStringCmd = redis.NewStringCmd(context.Background())
	staticBoolCmd   = redis.NewBoolCmd(context.Background())
)

type mockRedisClient struct {
	redis.UniversalClient
}

type mockPipeliner struct {
	redis.Pipeliner
	incrCmd  *redis.IntCmd
	doCmd    *redis.Cmd
	xaddCmds []*redis.StringCmd
	xaddFail bool
}

func (m *mockPipeliner) Incr(ctx context.Context, key string) *redis.IntCmd {
	m.incrCmd.SetVal(1)
	return m.incrCmd
}

func (m *mockPipeliner) Do(ctx context.Context, args ...any) *redis.Cmd {
	return m.doCmd
}

func (m *mockPipeliner) XAdd(ctx context.Context, args *redis.XAddArgs) *redis.StringCmd {
	cmd := redis.NewStringCmd(ctx)
	if m.xaddFail {
		cmd.SetErr(errors.New("fraud stream write failed"))
	} else {
		cmd.SetVal("1-0")
	}
	m.xaddCmds = append(m.xaddCmds, cmd)
	return cmd
}

func (m *mockPipeliner) Exec(ctx context.Context) ([]redis.Cmder, error) {
	for _, cmd := range m.xaddCmds {
		if err := cmd.Err(); err != nil {
			m.xaddCmds = m.xaddCmds[:0]
			return nil, err
		}
	}
	m.xaddCmds = m.xaddCmds[:0]
	return nil, nil
}

func (m *mockRedisClient) Pipeline() redis.Pipeliner {
	return &mockPipeliner{
		incrCmd: redis.NewIntCmd(context.Background()),
		doCmd:   redis.NewCmd(context.Background()),
	}
}

func (m *mockRedisClient) Set(ctx context.Context, key string, value any, expiration time.Duration) *redis.StatusCmd {
	return staticStatusCmd
}

func (m *mockRedisClient) Get(ctx context.Context, key string) *redis.StringCmd {
	staticStringCmd.SetVal("1716223400000")
	return staticStringCmd
}

func (m *mockRedisClient) Eval(ctx context.Context, script string, keys []string, args ...any) *redis.Cmd {
	staticCmd.SetVal(int64(0))
	return staticCmd
}

func (m *mockRedisClient) EvalSha(ctx context.Context, sha1 string, keys []string, args ...any) *redis.Cmd {
	staticCmd.SetVal(int64(0))
	return staticCmd
}

func (m *mockRedisClient) Process(ctx context.Context, cmd redis.Cmder) error {
	if c, ok := cmd.(*redis.Cmd); ok {
		c.SetVal(int64(0))
	}
	return nil
}

func setProcessLuaInt64(cmd redis.Cmder, v int64) {
	if c, ok := cmd.(*redis.Cmd); ok {
		c.SetVal(v)
	}
}

func setProcessLuaErr(cmd redis.Cmder, err error) {
	cmd.SetErr(err)
}

func (m *mockRedisClient) ScriptLoad(ctx context.Context, script string) *redis.StringCmd {
	staticStringCmd.SetVal("d3b07384d113edec49eaa6238ad5ff00")
	return staticStringCmd
}

func (m *mockRedisClient) SetNX(ctx context.Context, key string, value any, expiration time.Duration) *redis.BoolCmd {
	staticBoolCmd.SetVal(true)
	return staticBoolCmd
}

func (m *mockRedisClient) HExists(ctx context.Context, key string, field string) *redis.BoolCmd {
	staticBoolCmd.SetVal(false)
	return staticBoolCmd
}
