package client

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/Yunpeng-J/fabric-protos-go/common"
	"github.com/Yunpeng-J/fabric-protos-go/orderer"
	"github.com/Yunpeng-J/fabric-protos-go/peer"
	"github.com/Yunpeng-J/tape/pkg/metrics"
	"github.com/Yunpeng-J/tape/pkg/workload"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var logger *log.Logger

func init() {
	logger = log.New()
	logger.SetLevel(log.TraceLevel)
}

type ClientManager struct {
	clients   [][]*Client
	generator workload.Generator
	e2eCh     chan *Tracker
	metrics   *Metrics
}

func NewClientManager(e2eCh chan *Tracker, endorsers []Node, orderer Node, crypto *Crypto, client int, gen workload.Generator, provider metrics.Provider) *ClientManager {
	if client < 1 {
		panic("clientsPerEndorser must be greater than 0")
	}
	clientManager := &ClientManager{
		clients:   make([][]*Client, len(endorsers)),
		generator: gen,
		e2eCh:     e2eCh,
		metrics:   NewMetrics(provider),
	}
	cnt := 0
	for i := 0; i < len(endorsers); i++ {
		for j := 0; j < client; j++ {
			cli := NewClient(cnt, i, endorsers[i], orderer, crypto, clientManager.metrics, e2eCh)
			clientManager.clients[i] = append(clientManager.clients[i], cli)
			cnt += 1
		}
	}
	return clientManager
}
func (cm *ClientManager) Drain() {
	e2e := map[string]time.Time{}
	for {
		item := <-cm.e2eCh
		if it, ok := e2e[item.txid]; ok {
			cm.metrics.CommittedTransaction.Add(1)
			cm.metrics.E2eLatency.Observe(item.timestamp.Sub(it).Seconds())
			delete(e2e, item.txid)
		} else {
			e2e[item.txid] = item.timestamp
		}
	}
}

func (cm *ClientManager) Run(ctx context.Context) {
	go cm.Drain()
	for i := 0; i < len(cm.clients); i++ {
		for j := 0; j < len(cm.clients[i]); j++ {
			go cm.clients[i][j].StartDraining()
			go cm.clients[i][j].Run(cm.generator, ctx)
		}
	}
}

type Client struct {
	id         int
	endorserID int
	endorser   peer.EndorserClient
	orderer    orderer.AtomicBroadcast_BroadcastClient
	crypto     *Crypto
	e2eCh      chan *Tracker

	metrics *Metrics
}

func NewClient(id int, endorserId int, endorser, orderer Node, crypto *Crypto, metrics *Metrics, e2eCh chan *Tracker) *Client {
	client := &Client{
		id:         id,
		endorserID: endorserId,
		crypto:     crypto,
		metrics:    metrics,
		e2eCh:      e2eCh,
	}
	var err error
	client.orderer, err = CreateBroadcastClient(orderer, logger)
	if err != nil {
		panic(fmt.Sprintf("create ordererClient failed: %v", err))
	}
	client.endorser, err = CreateEndorserClient(endorser, logger)
	if err != nil {
		panic(fmt.Sprintf("create endorserClient failed: %v", err))
	}
	return client
}

func (client *Client) StartDraining() {
	// TODO: metric
	for {
		res, err := client.orderer.Recv()
		if err != nil {
			if err == io.EOF {
				return
			}
			logger.Errorf("recv from orderer error: %v", err)
			return
		}
		if res.Status != common.Status_SUCCESS {
			logger.Errorf("recv from orderer, status: %s", res.Status)
			continue
		}
	}
}

func (client *Client) Run(gen workload.Generator, ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			// timeout
			txn := gen.Stop()
			logger.Infof("client %d for endorser %d is ready to stop", client.id, client.endorserID)
			client.sendTransaction(txn)
			return
		default:
			// time.Sleep(10 * time.Millisecond)
			txn := gen.Workload()
			if len(txn) == 0 {
				// end of file
				txn := gen.Stop()
				client.sendTransaction(txn)
				return
			}
			client.sendTransaction(txn)
		}
	}
}

func (client *Client) sendTransaction(txn []string) (err error) {
	// benchmark
	start := time.Now()
	var endorsementLatency float64
	defer func() {
		if err != nil {
			logger.Errorf("send transaction failed: %v", err)
			// failed
		} else {
			client.metrics.EndorsementLatency.Observe(endorsementLatency)
			client.metrics.NumOfTransaction.Add(1)
			// async ordering latency
		}
	}()
	prop, txid, err := CreateProposal(
		txn[0],
		client.crypto,
		viper.GetString("channel"),
		viper.GetString("chaincode"),
		viper.GetString("version"),
		txn[1:],
	)
	if err != nil {
		logger.Errorf("create proposal failed: %v", err)
		return err
	}
	client.e2eCh <- &Tracker{
		txid:      txid,
		timestamp: start,
	}
	sprop, err := SignProposal(prop, client.crypto)
	if err != nil {
		logger.Errorf("sign proposal failed: %v", err)
		return err
	}
	r, err := client.endorser.ProcessProposal(context.Background(), sprop)
	if err != nil || r.Response.Status < 200 || r.Response.Status >= 400 {
		if r == nil {
			logger.Errorf("Err processing proposal: %s, status: unknown, endorser: %d \n", err, client.endorserID)
		} else {
			logger.Errorf("Err processing proposal: %s, status: %d, message: %s, addr: %d \n", err, r.Response.Status, r.Response.Message, client.endorserID)
		}
		return err
	}
	endorsementLatency = time.Since(start).Seconds()
	start = time.Now()
	envelope, err := CreateSignedTx(prop, client.crypto, []*peer.ProposalResponse{r}, viper.GetBool("checkRWSet"))
	if err != nil {
		logger.Errorf("create envelope %s failed: %v", txid, err)
		return err
	}
	err = client.orderer.Send(envelope)
	if err != nil {
		log.Errorf("send to order failed: %v", err)
		return err
	}
	return nil
}
