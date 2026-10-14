package cache

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Zapi-web/url-shortener/internal/domain"
	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
)

type Redis struct {
	rdb   redis.UniversalClient
	ttl   time.Duration
	ready atomic.Bool
}

type redisLogger struct{}

func (redisLogger) Printf(ctx context.Context, format string, v ...interface{}) {
	slog.DebugContext(ctx, "redis: "+fmt.Sprintf(format, v...))
}

var setLogger sync.Once

func NewRedis(ctx context.Context, addrs []string, masterName string, password string, ttl time.Duration) *Redis {
	rdb := redis.NewUniversalClient(&redis.UniversalOptions{
		Addrs:            addrs,
		MasterName:       masterName,
		Password:         password,
		SentinelPassword: password,
	})

	c := &Redis{
		ttl: ttl,
		rdb: rdb,
	}

	if err := redisotel.InstrumentTracing(rdb); err != nil {
		slog.WarnContext(ctx, "failed to initialize tracer in redis, continuing without it", "err", err)
	}

	if err := rdb.Ping(ctx).Err(); err != nil {
		slog.WarnContext(ctx, "redis is not reachable, keep trying", "err", err)
	} else {
		c.ready.Store(true)
	}

	setLogger.Do(func() { redis.SetLogger(redisLogger{}) })

	go c.healthloop(ctx)
	return c
}

func (c *Redis) healthloop(ctx context.Context) {
	t := time.NewTicker(2 * time.Second)
	defer t.Stop()

	var ok, fails int
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			pingCtx, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
			defer cancel()
			err := c.rdb.Ping(pingCtx).Err()
			if err == nil {
				fails = 0
				ok++
				if ok >= 2 && !c.ready.Load() {
					c.ready.Store(true)
					slog.InfoContext(ctx, "redis is available, enabling cache")
				}
			} else {
				ok = 0
				fails++
				if fails >= 3 && c.ready.Load() {
					c.ready.Store(false)
					slog.ErrorContext(ctx, "redis is unreachable, degraded to db-only mode", "err", err)
				}
			}
		}
	}
}

func (r *Redis) Set(ctx context.Context, shortURL string, longURL string) error {
	if shortURL == "" || longURL == "" {
		return domain.ErrInputisEmpty
	}
	if !r.ready.Load() {
		return domain.ErrCacheUnavailable
	}

	err := r.rdb.Set(ctx, shortURL, longURL, r.ttl).Err()

	if err != nil {
		return fmt.Errorf("failed to set key in cache: %w", err)
	}

	return nil
}

func (r *Redis) Get(ctx context.Context, shortURL string) (string, error) {
	if shortURL == "" {
		return "", domain.ErrInputisEmpty
	}
	if !r.ready.Load() {
		return "", domain.ErrCacheUnavailable
	}

	val, err := r.rdb.Get(ctx, shortURL).Result()

	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", domain.ErrUrlNotFound
		}

		return "", fmt.Errorf("failed to get key from cache: %w", err)
	}

	return val, nil
}

func (r *Redis) Close() error {
	return r.rdb.Close()
}
