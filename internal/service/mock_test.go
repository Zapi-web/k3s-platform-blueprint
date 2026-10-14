package service

import (
	"context"
	"time"

	"github.com/Zapi-web/url-shortener/internal/domain"
)

type mockCache struct {
	setFunc func(ctx context.Context, shortURL string, longURL string) error
	getFunc func(ctx context.Context, shortURL string) (string, error)
}

func (m *mockCache) Set(ctx context.Context, shortURL string, longURL string) error {
	if m.setFunc != nil {
		return m.setFunc(ctx, shortURL, longURL)
	}
	return nil
}

func (m *mockCache) Get(ctx context.Context, shortURL string) (string, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, shortURL)
	}
	return "", nil
}

type mockDatabase struct {
	setFunc func(ctx context.Context, url *domain.URL) error
	getFunc func(ctx context.Context, id uint64) (domain.URL, error)
}

func (m *mockDatabase) Set(ctx context.Context, url *domain.URL) error {
	if m.setFunc != nil {
		return m.setFunc(ctx, url)
	}

	return nil
}

func (m *mockDatabase) Get(ctx context.Context, id uint64) (domain.URL, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, id)
	}

	return domain.URL{}, nil
}

type mockEncoder struct {
	encodeFunc func(uint64) string
	decodeFunc func(string) (uint64, error)
}

func (m *mockEncoder) Encode(num uint64) string {
	if m.encodeFunc != nil {
		return m.encodeFunc(num)
	}
	return ""
}

func (m *mockEncoder) Decode(str string) (uint64, error) {
	if m.decodeFunc != nil {
		return m.decodeFunc(str)
	}
	return 0, nil
}

type mockKGS struct {
	generateFunc func() uint64
}

func (m *mockKGS) Generate() uint64 {
	if m.generateFunc != nil {
		return m.generateFunc()
	}

	return 0
}

type mockMetrics struct {
	incTotalCacheRequestFunc func(cacheStatus domain.CacheStatus)
	observeQueryDurationFunc func(handler string, duration time.Duration)
	incUrlsCreated           func()
	incInFlight              func(handler string)
	decInFlight              func(handler string)
}

func (m *mockMetrics) IncTotalCacheRequest(cacheStatus domain.CacheStatus) {
	if m.incTotalCacheRequestFunc != nil {
		m.incTotalCacheRequestFunc(cacheStatus)
	}
}

func (m *mockMetrics) ObserveQueryDuration(handler string, duration time.Duration) {
	if m.observeQueryDurationFunc != nil {
		m.observeQueryDurationFunc(handler, duration)
	}
}

func (m *mockMetrics) IncUrlsCreated() {
	if m.incUrlsCreated != nil {
		m.incUrlsCreated()
	}
}

func (m *mockMetrics) IncInFlight(handler string) {
	if m.incInFlight != nil {
		m.incInFlight(handler)
	}
}

func (m *mockMetrics) DecInFlight(handler string) {
	if m.decInFlight != nil {
		m.decInFlight(handler)
	}
}
