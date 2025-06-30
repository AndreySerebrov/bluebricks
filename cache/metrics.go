package cache

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type metrics struct {
	base        Fetcher
	summaryVect *prometheus.SummaryVec
}

func newMetrics(fetcher Fetcher) *metrics {
	return &metrics{
		base: fetcher,
		summaryVect: promauto.NewSummaryVec(
			prometheus.SummaryOpts{
				Name:       "cache_duration_seconds",
				Help:       "cache runtime duration and result",
				MaxAge:     time.Minute,
				Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
			},
			[]string{"result"}),
	}
}

func (m *metrics) Fetch(id int) (string, error) {
	var err error
	start := time.Now()
	defer func() {
		result := "success"
		duration := time.Since(start).Seconds()
		if err != nil {
			result = "error"
		}
		m.summaryVect.WithLabelValues(result).Observe(duration)
	}()
	data, err := m.base.Fetch(id)
	if err != nil {
		return "", err
	}
	// Process the fetched data as needed
	return data, nil
}
