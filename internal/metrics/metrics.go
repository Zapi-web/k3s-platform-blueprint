package metrics

import (
	"fmt"
	"net/http"
	"time"

	"github.com/VictoriaMetrics/metrics"
	"github.com/Zapi-web/url-shortener/internal/domain"
)

type VictoriaMetrics struct{}

func NewVM() *VictoriaMetrics {
	return &VictoriaMetrics{}
}

func (m *VictoriaMetrics) IncTotalRequests(handler string, status int) {
	name := fmt.Sprintf(`requests_total{handler="%s",status="%d"}`, handler, status)
	metrics.GetOrCreateCounter(name).Inc()
}

func (m *VictoriaMetrics) IncUrlsCreated() {
	metrics.GetOrCreateCounter("urls_created_total").Inc()
}

func (m *VictoriaMetrics) IncInFlight(handler string) {
	name := fmt.Sprintf(`in_flight_requests{handler="%s"}`, handler)
	metrics.GetOrCreateGauge(name, nil).Inc()
}

func (m *VictoriaMetrics) DecInFlight(handler string) {
	name := fmt.Sprintf(`in_flight_requests{handler="%s"}`, handler)
	metrics.GetOrCreateGauge(name, nil).Dec()
}

func (m *VictoriaMetrics) ObserveRequestDuration(handler string, status int, duration time.Duration) {
	name := fmt.Sprintf(`request_durations_seconds{handler="%s",status="%d"}`, handler, status)
	metrics.GetOrCreateHistogram(name).Update(duration.Seconds())
}

func (m *VictoriaMetrics) IncTotalCacheRequest(cacheStatus domain.CacheStatus) { // miss,hit,error,unavailable
	name := fmt.Sprintf(`cache_requests_total{cache_status="%s"}`, cacheStatus)
	metrics.GetOrCreateCounter(name).Inc()
}

func (m *VictoriaMetrics) ObserveQueryDuration(handler string, duration time.Duration) {
	name := fmt.Sprintf(`query_duration_seconds{handler="%s"}`, handler)
	metrics.GetOrCreateHistogram(name).Update(duration.Seconds())
}

func ExposeMetrics(w http.ResponseWriter, req *http.Request) {
	metrics.WritePrometheus(w, true)
}
