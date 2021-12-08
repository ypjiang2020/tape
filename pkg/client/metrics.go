package client

import "github.com/Yunpeng-J/tape/pkg/metrics"

var (
	endorsementLatency = metrics.HistogramOpts{
		Name:       "endorsementLatency",
		Help:       "from create transaction to receive endorsement",
		LabelNames: []string{"EndorserID", "ClientID"},
	}
	e2eLatency = metrics.HistogramOpts{
		Name: "e2eLatency",
		Help: "from create transaction to receive commit ack",
	}
	orderingLatency = metrics.HistogramOpts{
		Name:       "orderingLatency",
		Help:       "from broadcasting envelope to receiving ack",
		LabelNames: []string{"EndorserID", "ClientID"},
	}
	numOfTransaction = metrics.CounterOpts{
		Name:       "numOfTransaction",
		Help:       "the number of transaction that broadcast to ordering service",
		LabelNames: []string{"EndorserID", "ClientID"},
	}
	committedTransaction = metrics.CounterOpts{
		Name: "committedTransaction",
		Help: "the number of committed transaction",
	}
)

type Metrics struct {
	EndorsementLatency   metrics.Histogram
	E2eLatency           metrics.Histogram
	OrderingLatency      metrics.Histogram
	NumOfTransaction     metrics.Counter
	CommittedTransaction metrics.Counter
}

func NewMetrics(p metrics.Provider) *Metrics {
	return &Metrics{
		EndorsementLatency:   p.NewHistogram(endorsementLatency),
		E2eLatency:           p.NewHistogram(e2eLatency),
		OrderingLatency:      p.NewHistogram(orderingLatency),
		NumOfTransaction:     p.NewCounter(numOfTransaction),
		CommittedTransaction: p.NewCounter(committedTransaction),
	}
}
