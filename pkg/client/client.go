package client

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Yunpeng-J/fabric-protos-go/common"
	"github.com/Yunpeng-J/fabric-protos-go/peer"
	"github.com/Yunpeng-J/tape/pkg/workload"
	"github.com/Yunpeng-J/tape/pkg/workload/smallbank"
	"github.com/Yunpeng-J/tape/pkg/workload/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var logger *log.Logger

func init() {
	logger = log.New()
	logger.SetLevel(log.TraceLevel)
}

type ClientManager struct {
	clients            [][]*Client
	broadcaster        []*Broadcaster
	workload           workload.Provider
	e2eCh              chan *Tracker
	client2broadcaster chan *common.Envelope
	metrics            *Metrics
	resubmit           chan string
	resubmitFlag       bool
}

func NewClientManager(e2eCh chan *Tracker, endorsers []Node, orderer Node, crypto *Crypto, client int, gen workload.Provider, metric *Metrics, resub chan string) *ClientManager {
	if client < 1 {
		panic("clientsPerEndorser must be greater than 0")
	}
	broadcasterNum := viper.GetInt("broadcasterNum")
	clientManager := &ClientManager{
		clients:            make([][]*Client, len(endorsers)),
		broadcaster:        make([]*Broadcaster, broadcasterNum),
		workload:           gen,
		e2eCh:              e2eCh,
		client2broadcaster: make(chan *common.Envelope, 100000),
		metrics:            metric,
		resubmit:           resub,
	}
	val, _ := os.LookupEnv("RESUBMIT")
	clientManager.resubmitFlag = val == "true"
	for i := 0; i < broadcasterNum; i++ {
		log.Println("create orderer client")
		clientManager.broadcaster[i] = NewBroadcaster(i, orderer, clientManager.client2broadcaster)
	}
	cnt := 0
	for i := 0; i < len(endorsers); i++ {
		session := utils.GetName(20)
		for j := 0; j < client; j++ {
			generator := clientManager.workload.ForEachClient(cnt, session)
			log.Println("create endorser client")
			cli := NewClient(clientManager, cnt, i, endorsers[i], crypto, generator, clientManager.metrics, e2eCh, clientManager.client2broadcaster)
			clientManager.clients[i] = append(clientManager.clients[i], cli)
			cnt += 1
		}
	}
	if viper.GetString("transactionType") == "txn" {
		go gen.Start()
	}
	logger.Infof("endorsers: %d clients: %d", len(endorsers), cnt)
	return clientManager
}

func (cm *ClientManager) Execute(node int, txn []string) string {
	return cm.clients[node][0].Execute(txn)
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

func (cm *ClientManager) Run(ctx context.Context, done chan struct{}) {
	go cm.Drain()
	for i := 0; i < len(cm.clients); i++ {
		for j := 0; j < len(cm.clients[i]); j++ {
			go cm.clients[i][j].Run(ctx, done)
		}
	}
	for i := 0; i < len(cm.broadcaster); i++ {
		go cm.broadcaster[i].Run(done)
	}
}

type Client struct {
	cm                 *ClientManager
	id                 string
	endorserID         string
	workload           smallbank.GeneratorT
	endorser           peer.EndorserClient
	crypto             *Crypto
	e2eCh              chan *Tracker
	done               chan struct{}
	client2broadcaster chan *common.Envelope

	metrics *Metrics
}

func NewClient(cm *ClientManager, id int, endorserId int, endorser Node, crypto *Crypto, generate smallbank.GeneratorT, metrics *Metrics, e2eCh chan *Tracker, c2b chan *common.Envelope) *Client {
	client := &Client{
		cm:                 cm,
		id:                 strconv.Itoa(id),
		endorserID:         strconv.Itoa(endorserId),
		workload:           generate,
		crypto:             crypto,
		metrics:            metrics,
		e2eCh:              e2eCh,
		done:               make(chan struct{}),
		client2broadcaster: c2b,
	}
	var err error
	client.endorser, err = CreateEndorserClient(endorser, logger)
	if err != nil {
		panic(fmt.Sprintf("create endorserClient failed: %v", err))
	}
	return client
}

func (client *Client) Execute(txn []string) string {
	prop, _, err := CreateProposal(
		utils.GetName(64),
		client.crypto,
		viper.GetString("channel"),
		viper.GetString("chaincode"),
		viper.GetString("version"),
		txn,
	)
	if err != nil {
		logger.Errorf("create proposal failed: %v", err)
		return fmt.Sprintf("%v", err)
	}
	sprop, err := SignProposal(prop, client.crypto)
	if err != nil {
		logger.Errorf("sign proposal failed: %v", err)
		return fmt.Sprintf("%v", err)
	}
	r, err := client.endorser.ProcessProposal(context.Background(), sprop)
	if err != nil || r.Response.Status < 200 || r.Response.Status >= 400 {
		if r == nil {
			logger.Errorf("Err processing proposal: %v, status: unknown, endorser: %s \n", err, client.endorserID)
		} else {
			logger.Errorf("Err processing proposal: %v, status: %d, message: %s, addr: %s \n", err, r.Response.Status, r.Response.Message, client.endorserID)
		}
		return fmt.Sprintf("%v", err)
	}
	return string(r.Payload)
}

func (client *Client) Run(ctx context.Context, done chan struct{}) {
	defer func() {
		client.done <- struct{}{}
	}()
	for {
		select {
		case <-ctx.Done():
			// timeout
			txn := client.workload.Stop()
			logger.Infof("client %s for endorser %s is ready to stop", client.id, client.endorserID)
			client.sendTransaction(txn)
			return
		case <-done:
			return
		default:
			// time.Sleep(10 * time.Millisecond)
			txn := client.workload.Generate()
			if len(txn) == 0 {
				// end of file
				txn := client.workload.Stop()
				client.sendTransaction(txn)
				return
			}
			client.sendTransaction(txn)
		}
	}
}

func (client *Client) sendTransaction(txn []string) (err error) {
	// log.Println("client send", client.id, txn)
	// benchmark
	start := time.Now()
	var endorsementLatency float64
	defer func() {
		if err != nil {
			// logger.Errorf("send transaction failed: %v", err)
			// failed

			// resubmit
			if client.cm.resubmitFlag {
				temp := strings.Split(txn[0], "_+=+_")
				client.cm.resubmit <- temp[2]
			}

		} else {
			client.metrics.EndorsementLatency.With("EndorserID", client.endorserID, "ClientID", client.id).Observe(endorsementLatency)
			client.metrics.NumOfTransaction.With("EndorserID", client.endorserID, "ClientID", client.id).Add(1)
			client.metrics.OrderingLatency.With("EndorserID", client.endorserID, "ClientID", client.id).Observe(time.Since(start).Seconds())
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
			logger.Errorf("Err processing proposal: %v, status: unknown, endorser: %s latency=%d\n", err, client.endorserID, time.Since(start).Seconds())
		} else {
			logger.Errorf("Err processing proposal: %v, status: %d, message: %s, addr: %s \n", err, r.Response.Status, r.Response.Message, client.endorserID)
		}
		return err
	}
	// log.Println("client receive", client.id, txn)
	endorsementLatency = time.Since(start).Seconds()
	start = time.Now()
	envelope, err := CreateSignedTx(prop, client.crypto, []*peer.ProposalResponse{r}, viper.GetBool("checkRWSet"))
	if err != nil {
		logger.Errorf("create envelope %s failed: %v", txid, err)
		return err
	}
	client.client2broadcaster <- envelope
	// log.Println("len of client2broadcaster", len(client.client2broadcaster))
	return nil
}
