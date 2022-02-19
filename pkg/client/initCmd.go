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
	seed := int64(viper.GetInt("newseed"))
	if seed == 0 {
		seed = time.Now().UnixNano()
	}
	rand.Seed(seed)
	start := time.Now()
	metricsSystem = operations.NewSystem(operations.Options{
		ListenAddress: viper.GetString("metricsAddr"),
		Provider:      viper.GetString("metricsType"),
	})
	metric := NewMetrics(metricsSystem.Provider)
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
		metric,
	)
	done := make(chan struct{})
	resub := make(chan string, 10000)
	viper.SetDefault("shardNumber", len(config.Endorsers))
	viper.SetDefault("clientsNumber", len(config.Endorsers)*viper.GetInt("clientsPerEndorser"))

	workload := workload.NewWorkloadProvider(metricsSystem.Provider, resub)
	cm := NewClientManager(
		e2eCh,
		config.Endorsers,
		config.Orderer,
		crypto,
		viper.GetInt("clientsPerEndorser"),
		workload.Provider,
		metric,
		resub,
	)
	timeout := viper.GetInt("interval")
	if viper.GetString("transactionType") == "init" {
		timeout = 10000
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	_ = cancel
	cm.Run(ctx, done)
	go observer.Start(viper.GetInt("clientsPerEndorser")*len(config.Endorsers), resub, done)
	for {
		select {
		case <-done:
			logger.Infof("finish RunInitCmd in %d ms", time.Since(start).Milliseconds())
			if viper.GetBool("wait") == true {
				time.Sleep(time.Duration(100000) * time.Second)
			}
			return
		}
	}
}
