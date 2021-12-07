package workload

import (
	"fmt"
	"github.com/Yunpeng-J/tape/pkg/workload/smallbank"
	"github.com/spf13/viper"
)

type Generator interface {
	Generate() []string
	Stop() []string
}

type Provider interface {
	ForEachClient(i int) Generator
}


type WorkloadProvider struct {
	Provider
}

func NewWorkloadProvider() *WorkloadProvider {
	wlp := &WorkloadProvider{}
	workload := viper.GetString("workload")
	switch workload {
	case "smallbank":
		wlp.Provider = smallbank.NewSmallBank()
	case "kv":
		panic("TODO")
	default:
		panic(fmt.Sprintf("Unknown workload type: %s", workload))
	}
	return wlp
}
