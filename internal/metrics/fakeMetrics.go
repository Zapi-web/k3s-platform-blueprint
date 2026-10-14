package metrics

import (
	"time"

	"github.com/Zapi-web/url-shortener/internal/domain"
)

type FakeMetrics struct{}

func NewFake() *FakeMetrics {
	return &FakeMetrics{}
}

func (m *FakeMetrics) IncTotalRequests(handler string, status int)                               {}
func (m *FakeMetrics) ObserveRequestDuration(handler string, status int, duration time.Duration) {}
func (m *FakeMetrics) IncUrlsCreated()                                                           {}
func (m *FakeMetrics) IncInFlight(handler string)                                                {}
func (m *FakeMetrics) DecInFlight(handler string)                                                {}
func (m *FakeMetrics) IncTotalCacheRequest(domain.CacheStatus)                                   {}
func (m *FakeMetrics) ObserveQueryDuration(handler string, duration time.Duration)               {}
