package fraud

import (
	"context"
	"errors"

	redis "github.com/redis/go-redis/v9"
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

func (p *mockPipeliner) XAdd(ctx context.Context, args *redis.XAddArgs) *redis.StringCmd {
	cmd := redis.NewStringCmd(ctx)
	if p.xaddFail {
		cmd.SetErr(errors.New("fraud stream write failed"))
	} else {
		cmd.SetVal("1-0")
	}
	p.xaddCmds = append(p.xaddCmds, cmd)
	return cmd
}

func (p *mockPipeliner) Incr(ctx context.Context, key string) *redis.IntCmd {
	if p.incrCmd != nil {
		return p.incrCmd
	}
	return redis.NewIntCmd(ctx)
}

func (p *mockPipeliner) Do(ctx context.Context, args ...interface{}) *redis.Cmd {
	if p.doCmd != nil {
		return p.doCmd
	}
	return redis.NewCmd(ctx)
}

func (p *mockPipeliner) Exec(ctx context.Context) ([]redis.Cmder, error) {
	for _, cmd := range p.xaddCmds {
		if err := cmd.Err(); err != nil {
			p.xaddCmds = p.xaddCmds[:0]
			return nil, err
		}
	}
	p.xaddCmds = p.xaddCmds[:0]
	return nil, nil
}

func (m *mockRedisClient) Pipeline() redis.Pipeliner {
	return &mockPipeliner{
		incrCmd: redis.NewIntCmd(context.Background()),
		doCmd:   redis.NewCmd(context.Background()),
	}
}
