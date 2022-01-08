package client

import (
	"strconv"
	"strings"
	"time"

	"github.com/Yunpeng-J/fabric-protos-go/peer"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type Observer struct {
	d      peer.Deliver_DeliverFilteredClient
	e2eCh  chan *Tracker
	logger *log.Logger

	resubmits map[string]int

	metrics *Metrics
}

func NewObserver(channel string, node Node, crypto *Crypto, logger *log.Logger, e2eCh chan *Tracker, metric *Metrics) (*Observer, error) {
	deliverer, err := CreateDeliverFilteredClient(node, logger)
	if err != nil {
		return nil, err
	}

	seek, err := CreateSignedDeliverNewestEnv(channel, crypto)
	if err != nil {
		return nil, err
	}

	if err = deliverer.Send(seek); err != nil {
		return nil, err
	}

	// drain first response
	if _, err = deliverer.Recv(); err != nil {
		return nil, err
	}

	return &Observer{d: deliverer, logger: logger, e2eCh: e2eCh, resubmits: map[string]int{}, metrics: metric}, nil
}

func (o *Observer) Start(numOfClients int, resub chan string, done chan struct{}) {
	o.logger.Debugf("start observer")
	cnt := viper.GetInt("transactionNumber")
	retry := viper.GetInt("retry")
	if viper.GetString("transactionType") == "init" {
		cnt = viper.GetInt("accountNumber")
	}
	quit := cnt
	defer func() {
		log.Println("transactionNumber", cnt)
		log.Println("retry", retry)
		log.Println("quit", quit)
		o.PrintInfo()
		close(done)
	}()
	resubmitFlag := viper.GetBool("resubmit")
	for {
		if quit <= 0 {
			if !resubmitFlag {
				return
			}
		}
		r, err := o.d.Recv()
		if err != nil {
			o.logger.Errorf("observer receive from committer error: %v", err)
			return
		}
		if r == nil {
			o.logger.Panicf("received nil message, but expect a valid block instead. You could look into your peer logs for more info")
			return
		}
		cur := time.Now()
		switch t := r.Type.(type) {
		case *peer.DeliverResponse_FilteredBlock:
			txns := len(t.FilteredBlock.FilteredTransactions)
			commits := 0
			o.logger.Infof("observer receive block %d with length %d", t.FilteredBlock.Number, txns)
			for _, tx := range t.FilteredBlock.FilteredTransactions {
				quit -= 1
				txid := tx.GetTxid()
				o.e2eCh <- &Tracker{
					txid:      txid,
					timestamp: cur,
				}
				temp := strings.Split(txid, "_+=+_")
				// log.Printf("txid %v", temp)
				o.resubmits[temp[2]] += 1
				if tx.TxValidationCode == peer.TxValidationCode_VALID {
					o.metrics.NumOfCommits.Add(1)
					commits += 1
					cnt -= 1
					if cnt == 0 {
						log.Println("all transactions finished")
						return
					}
				} else if tx.TxValidationCode == peer.TxValidationCode_MVCC_READ_CONFLICT {
					o.metrics.NumOfAborts.Add(1)
					if resubmitFlag {
						resub <- temp[2]
					}
					retry -= 1
					if retry == 0 {
						log.Println("retry run out, but still cannot commit all transactions. quit")
						return

					}
				} else {
					o.logger.Errorf("transaction error: %s", tx.TxValidationCode)
				}
				// if txid[len(txid)-5:] == "#end#" {
				// 	numOfClients -= 1
				// 	o.logger.Infof("some client ends")
				// 	if numOfClients == 0 {
				// 		o.logger.Infof("observer finished")
				// 		return
				// 	}
				// }
			}
			o.metrics.AbortRatePerBlock.With("blockid", strconv.Itoa(int(t.FilteredBlock.Number))).Add(float64(txns-commits) / float64(txns))
		case *peer.DeliverResponse_Status:
			o.logger.Infof("observer receive from orderer, status:", t.Status)
		default:
			o.logger.Errorf("observer receive from orderer: unknown. Please check the return value manually")
		}
	}
}

func (o *Observer) PrintInfo() {
	for k, v := range o.resubmits {
		log.Printf("resubmit %s %d", k, v)
	}
}
