package service

import (
	"context"
	"time"

	"github.com/Zapi-web/url-shortener/internal/domain"
)

type Cache interface {
	Set(ctx context.Context, shortURL string, longURL string) error
	Get(ctx context.Context, shortURL string) (string, error)
}

type Database interface {
	Set(ctx context.Context, url *domain.URL) error
	Get(ctx context.Context, id uint64) (domain.URL, error)
}

type Encoder interface {
	Encode(uint64) string
	Decode(string) (uint64, error)
}

type KGS interface {
	Generate() uint64
}

type Metrics interface {
	IncTotalCacheRequest(domain.CacheStatus)
	ObserveQueryDuration(handler string, duration time.Duration)
	IncUrlsCreated()
	IncInFlight(handler string)
	DecInFlight(handler string)
}
