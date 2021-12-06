package operations

import (
	"fmt"
	"github.com/Yunpeng-J/tape/pkg/metrics"
	"github.com/Yunpeng-J/tape/pkg/metrics/disabled"
	"github.com/Yunpeng-J/tape/pkg/metrics/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
)

type Options struct {
	ListenAddress string
	Provider string
}

type System struct {
	metrics.Provider
	options Options
}

func NewSystem(o Options) *System {
	system := &System{
		options: o,
	}
	system.initializeMetricsProvider()
	return system
}

func (s *System) initializeMetricsProvider() error {
	switch s.options.Provider {
	case "prometheus":
		s.Provider = &prometheus.Provider{}
		http.Handle("/metrics", promhttp.Handler())
		go http.ListenAndServe(s.options.ListenAddress, nil)
		return nil
	default:
		if s.options.Provider != "disabled" {
			panic(fmt.Sprintf("Unknown provider type: %s", s.options.Provider))
		}
		s.Provider = &disabled.Provider{}
		return nil
	}
}

