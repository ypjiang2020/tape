package client

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/Yunpeng-J/tape/pkg/workload"

	"github.com/Yunpeng-J/tape/pkg/operations"
	"github.com/spf13/viper"
)

var (
	metricsSystem *operations.System
	MAX_BUF       = 100010
)

func RunInitCmd(config Config) {
	runCmd(config)
}

func runCmd(config Config) {
	rand.Seed(int64(config.Seed))
	start := time.Now()
	metricsSystem = operations.NewSystem(operations.Options{
		ListenAddress: viper.GetString("metricsAddr"),
		Provider:      viper.GetString("metricsType"),
	})
	crypto, err := config.LoadCrypto()
	if err != nil {
		panic(fmt.Sprintf("load crypto failed: %v", err))
	}

	e2eCh := make(chan *Tracker, MAX_BUF)

	observer, err := NewObserver(
		viper.GetString("channel"),
		config.Committer,
		crypto,
		logger,
		e2eCh,
	)
	done := make(chan struct{})
	go observer.Start(viper.GetInt("clientsPerEndorser")*len(config.Endorsers), done)

	workload := workload.NewWorkloadProvider()
	cm := NewClientManager(
		e2eCh,
		config.Endorsers,
		config.Orderer,
		crypto,
		viper.GetInt("clientsPerEndorser"),
		workload.Provider,
		metricsSystem.Provider,
	)
	timeout := viper.GetInt("interval")
	if viper.GetString("transactionType") == "init" {
		timeout = 10000
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	_ = cancel
	cm.Run(ctx)
	for {
		select {
		case <-done:
			logger.Infof("finish RunInitCmd in %d ms", time.Since(start).Milliseconds())
			time.Sleep(time.Duration(1000) * time.Second)
			return
		}
	}
}
