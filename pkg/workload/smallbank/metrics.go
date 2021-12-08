package smallbank

import "github.com/Yunpeng-J/tape/pkg/metrics"

var (
	generatorLatency = metrics.HistogramOpts{
		Name:       "generatorLatencySmallbank",
		Help:       "create transaction",
		LabelNames: []string{"generator"},
	}
	generatorCnt = metrics.CounterOpts{
		Name:       "generatorSmallbank",
		Help:       "number of generator calls",
		LabelNames: []string{"generator"},
	}
	createCnt = metrics.CounterOpts{
		Name:       "createSmallbank",
		Help:       "",
		LabelNames: []string{"generator"},
	}
)

type Metrics struct {
	GeneratorCounter metrics.Counter
	CreateCounter    metrics.Counter
	GeneratorLatency metrics.Histogram
}

func NewMetrics(p metrics.Provider) *Metrics {
	return &Metrics{
		GeneratorCounter: p.NewCounter(generatorCnt),
		CreateCounter:    p.NewCounter(createCnt),
		GeneratorLatency: p.NewHistogram(generatorLatency),
	}
}
