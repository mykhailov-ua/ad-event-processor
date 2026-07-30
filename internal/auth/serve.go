package auth

import (
	"context"
	"log/slog"
	"net"
	"sync"
	"time"

	"espx/internal/auth/db"
	"espx/internal/auth/pb"
	"espx/internal/config"
	"espx/internal/database"
	"espx/pkg/lifecycle"

	"github.com/redis/go-redis/v9"
	google_grpc "google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func Serve(ctx context.Context, cfg *config.Config) error {
	if cfg == nil {
		return nil
	}

	pool, err := database.Connect(ctx, string(cfg.DBDSN), cfg.DBTrackerMaxConns, cfg.DBMinConns)
	if err != nil {
		return err
	}
	defer pool.Close()

	rdb, err := database.ConnectRedisShard(ctx, cfg, 0, database.RedisShardOptions{
		PoolSize: cfg.RedisPoolSize,
	})
	if err != nil {
		return err
	}
	defer rdb.Close()

	var controlRdbs []redis.UniversalClient
	controlRdbs, _, err = database.ConnectRedisShards(ctx, cfg, database.RedisShardOptions{
		PoolSize: cfg.RedisPoolSize,
	})
	if err != nil {
		slog.Warn("multi-shard redis unavailable, using single shard for control", "error", err)
		controlRdbs = []redis.UniversalClient{rdb}
	} else {
		defer func() {
			for _, shard := range controlRdbs {
				if shard != nil && shard != rdb {
					_ = shard.Close()
				}
			}
		}()
	}

	repo := db.NewStore(pool)
	tokenMaker, err := NewPasetoMaker(string(cfg.TokenSymmetricKey))
	if err != nil {
		return err
	}

	lockoutLimiter := NewLockoutLimiter(controlRdbs...)
	hasher, err := NewPasswordHasher(
		uint32(cfg.Argon2Memory),
		uint32(cfg.Argon2Iterations),
		uint8(cfg.Argon2Parallelism),
	)
	if err != nil {
		return err
	}

	authService := NewService(repo, tokenMaker, hasher, lockoutLimiter, rdb)
	authService.SetControlRedisShards(controlRdbs)
	cleanupWorker := NewSessionCleanupWorker(authService)

	workerCtx, workerCancel := context.WithCancel(ctx)
	var cleanupWG sync.WaitGroup
	cleanupWG.Add(1)
	go func() {
		defer cleanupWG.Done()
		cleanupWorker.Start(workerCtx, time.Minute)
	}()

	grpcHandler := NewHandler(authService, cfg)
	timeouts := lifecycle.TimeoutsFromConfig(cfg)

	lis, err := net.Listen("tcp", ":"+cfg.AuthServerPort)
	if err != nil {
		workerCancel()
		return err
	}

	server := google_grpc.NewServer()
	pb.RegisterAuthServiceServer(server, grpcHandler)
	if cfg.Env != "production" {
		reflection.Register(server)
	}

	metricsSrv := lifecycle.StartMetrics(":" + cfg.AuthMetricsPort)
	slog.Info("starting auth gRPC server", "port", cfg.AuthServerPort)

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- server.Serve(lis)
	}()

	select {
	case <-ctx.Done():
	case err := <-serveErr:
		if err != nil {
			workerCancel()
			return err
		}
	}

	workerCancel()
	if err := lifecycle.Wait(timeouts.Wait, cleanupWG.Wait); err != nil {
		slog.Warn("auth session cleanup worker drain timed out", "error", err)
	}
	lifecycle.ShutdownGRPC(server, timeouts.Shutdown)
	if err := metricsSrv.Shutdown(timeouts.Shutdown); err != nil {
		slog.Error("auth metrics server shutdown failed", "error", err)
	}
	return ctx.Err()
}
