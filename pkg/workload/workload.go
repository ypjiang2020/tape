package workload

import (
	"fmt"

	"github.com/Yunpeng-J/tape/pkg/metrics"
	"github.com/Yunpeng-J/tape/pkg/workload/smallbank"
	"github.com/spf13/viper"
)

type Provider interface {
	ForEachClient(i int) smallbank.GeneratorT
}

type WorkloadProvider struct {
	Provider
}

func NewWorkloadProvider(provider metrics.Provider) *WorkloadProvider {
	wlp := &WorkloadProvider{}
	workload := viper.GetString("workload")
	switch workload {
	case "smallbank":
		wlp.Provider = smallbank.NewSmallBank(provider)
	case "kv":
		panic("TODO")
	default:
		panic(fmt.Sprintf("Unknown workload type: %s", workload))
	}
	return wlp
}
