package client

import (
	"time"

	"github.com/Yunpeng-J/fabric-protos-go/peer"
	log "github.com/sirupsen/logrus"
)

type Observer struct {
	d      peer.Deliver_DeliverFilteredClient
	e2eCh  chan *Tracker
	logger *log.Logger

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

	return &Observer{d: deliverer, logger: logger, e2eCh: e2eCh, metrics: metric}, nil
}

func (o *Observer) Start(numOfClients int, done chan struct{}) {
	o.logger.Debugf("start observer")
	cnt := 0
	for {
		// o.logger.Infof("observer receiving %d", cnt)
		cnt += 1
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
			o.logger.Infof("observer receive block %d with length %d", t.FilteredBlock.Number, len(t.FilteredBlock.FilteredTransactions))
			for _, tx := range t.FilteredBlock.FilteredTransactions {
				txid := tx.GetTxid()
				o.e2eCh <- &Tracker{
					txid:      txid,
					timestamp: cur,
				}
				if tx.TxValidationCode == peer.TxValidationCode_VALID {
					o.metrics.NumOfCommits.Add(1)
				} else if tx.TxValidationCode == peer.TxValidationCode_MVCC_READ_CONFLICT {
					o.metrics.NumOfAborts.Add(1)
				} else {
					o.logger.Errorf("transaction error: %s", tx.TxValidationCode)
				}
				if txid[len(txid)-5:] == "#end#" {
					numOfClients -= 1
					o.logger.Infof("some client ends")
					if numOfClients == 0 {
						o.logger.Infof("observer finished")
						done <- struct{}{}
						return
					}
				}
			}
		case *peer.DeliverResponse_Status:
			o.logger.Infof("observer receive from orderer, status:", t.Status)
		default:
			o.logger.Errorf("observer receive from orderer: unknown. Please check the return value manually")
		}
	}
}
