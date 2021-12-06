package client

import "github.com/Yunpeng-J/tape/pkg/metrics"

var (
	endorsementLatency = metrics.HistogramOpts{
		Name: "endorsementLatency",
		Help: "from create transaction to receive endorsement",
	}
	e2eLatency = metrics.HistogramOpts{
		Name: "e2eLatency",
		Help: "from create transaction to receive commit ack",
	}
	numOfTransaction = metrics.CounterOpts{
		Name: "numOfTransaction",
		Help: "the number of transaction that broadcast to ordering service",
	}
	committedTransaction = metrics.CounterOpts{
		Name: "committedTransaction",
		Help: "the number of committed transaction",
	}
)

type Metrics struct {
	EndorsementLatency   metrics.Histogram
	E2eLatency           metrics.Histogram
	NumOfTransaction     metrics.Counter
	CommittedTransaction metrics.Counter
}

func NewMetrics(p metrics.Provider) *Metrics {
	return &Metrics{
		EndorsementLatency:   p.NewHistogram(endorsementLatency),
		E2eLatency:           p.NewHistogram(e2eLatency),
		NumOfTransaction:     p.NewCounter(numOfTransaction),
		CommittedTransaction: p.NewCounter(committedTransaction),
	}
}
