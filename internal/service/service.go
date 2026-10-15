package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/Zapi-web/url-shortener/internal/domain"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/singleflight"
)

type Service struct {
	database   Database
	cache      Cache
	encoder    Encoder
	kgs        KGS
	metrics    Metrics
	tracer     trace.Tracer
	defaultTTL time.Duration
	group      singleflight.Group
}

func New(db Database, cache Cache, kgs KGS, encoder Encoder, metrics Metrics, ttl time.Duration) *Service {
	return &Service{
		database:   db,
		cache:      cache,
		encoder:    encoder,
		kgs:        kgs,
		metrics:    metrics,
		tracer:     otel.Tracer("url-shortener/service"),
		defaultTTL: ttl,
	}
}

func (s *Service) Create(ctx context.Context, longURL string, userID uint64) (string, error) {
	s.metrics.IncInFlight("create")
	defer s.metrics.DecInFlight("create")

	ctx, span := s.tracer.Start(ctx, "Service.Create",
		trace.WithAttributes(attribute.Int64("user_id", int64(userID))),
	)
	defer span.End()

	if longURL == "" {
		return "", domain.ErrInputisEmpty
	}
	if userID == 0 {
		return "", domain.ErrInvalidInput
	}

	id := s.kgs.Generate()

	url := domain.URL{
		ID:        id,
		UserID:    userID,
		LongURL:   longURL,
		ExpiredAt: time.Now().UTC().Add(s.defaultTTL),
	}

	now := time.Now()
	err := s.database.Set(ctx, &url)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())

		slog.ErrorContext(ctx, "failed to save url to database", "id", id, "user_id", userID, "err", err)
		return "", fmt.Errorf("failed to set a url %d in database: %w", id, err)
	}

	s.metrics.ObserveQueryDuration("create", time.Since(now))
	slog.DebugContext(ctx, "saved url to database", "id", url.ID, "user_id", userID)

	encodedID := s.encoder.Encode(url.ID)

	s.cacheSet(ctx, encodedID, longURL)

	s.metrics.IncUrlsCreated()

	return encodedID, nil
}

func (s *Service) Get(ctx context.Context, shortURL string) (string, error) {
	s.metrics.IncInFlight("get")
	defer s.metrics.DecInFlight("get")

	ctx, span := s.tracer.Start(ctx, "Service.Get",
		trace.WithAttributes(attribute.String("short_url", shortURL)),
	)
	defer span.End()

	if shortURL == "" {
		return "", domain.ErrInputisEmpty
	}

	decodedID, err := s.encoder.Decode(shortURL)

	if err != nil {
		return "", domain.ErrInvalidInput
	}

	getCacheCtx, getCancel := context.WithTimeout(ctx, 200*time.Millisecond)
	defer getCancel()
	res, err := s.cache.Get(getCacheCtx, shortURL)

	if err == nil {
		slog.DebugContext(ctx, "retrieved url from cache", "key", shortURL)
		s.metrics.IncTotalCacheRequest(domain.CacheHit)
		return res, nil
	}

	switch {
	case errors.Is(err, domain.ErrUrlNotFound):
		s.metrics.IncTotalCacheRequest(domain.CacheMiss)
	case errors.Is(err, domain.ErrCacheUnavailable):
		s.metrics.IncTotalCacheRequest(domain.CacheUnavailable)
	default:
		slog.WarnContext(ctx, "cache lookup failed, falling back to database", "key", shortURL, "err", err)
		s.metrics.IncTotalCacheRequest(domain.CacheError)
	}

	start := time.Now()

	dbRes, err, _ := s.group.Do(shortURL, func() (any, error) {
		dbCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 1*time.Second)
		defer cancel()

		return s.database.Get(dbCtx, decodedID)
	})

	s.metrics.ObserveQueryDuration("get", time.Since(start))

	if err != nil {
		if errors.Is(err, domain.ErrUrlNotFound) {
			slog.DebugContext(ctx, "url not found in database", "short_url", shortURL, "decoded_id", decodedID)
			return "", domain.ErrUrlNotFound
		}

		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		slog.ErrorContext(ctx, "failed to query database for short url", "short_url", shortURL, "decoded_id", decodedID, "err", err)
		return "", fmt.Errorf("failed to find short url in database: %w", err)
	}

	url, ok := dbRes.(domain.URL)
	if !ok {
		return "", fmt.Errorf("unexpected type %T received from database", dbRes)
	}

	s.cacheSet(ctx, shortURL, url.LongURL)

	return url.LongURL, nil
}

func (s *Service) cacheSet(ctx context.Context, key, value string) {
	cacheCtx, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
	defer cancel()
	err := s.cache.Set(cacheCtx, key, value)

	if err != nil {
		switch {
		case errors.Is(err, domain.ErrCacheUnavailable):
			s.metrics.IncTotalCacheRequest(domain.CacheUnavailable)
		default:
			slog.ErrorContext(ctx, "unknown error occurred when sending key-value to cache", "err", err)
			s.metrics.IncTotalCacheRequest(domain.CacheError)
		}
	} else {
		slog.DebugContext(ctx, "set key in cache", "key", key)
	}
}
