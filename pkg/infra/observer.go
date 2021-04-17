package infra

import (
	"time"

	"github.com/hyperledger/fabric-protos-go/peer"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type Observer struct {
	d      peer.Deliver_DeliverFilteredClient
	logger *log.Logger
}

func CreateObserver(channel string, node Node, crypto *Crypto, logger *log.Logger) (*Observer, error) {
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

	return &Observer{d: deliverer, logger: logger}, nil
}

func (o *Observer) Start(N int32, errorCh chan error, finishCh chan struct{}, now time.Time, abort *int32) {
	defer close(finishCh)
	o.logger.Debugf("start observer")
	var n int32 = 0
	for n+(*abort) < N {
		r, err := o.d.Recv()
		if err != nil {
			errorCh <- err
		}

		if r == nil {
			errorCh <- errors.Errorf("received nil message, but expect a valid block instead. You could look into your peer logs for more info")
			return
		}

		fb := r.Type.(*peer.DeliverResponse_FilteredBlock)
		n = n + int32(len(fb.FilteredBlock.FilteredTransactions))
		st := time.Now().UnixNano()
		g_block = append(g_block, fb)
		g_timestampe = append(g_timestampe, st)
		// for _, tx := range fb.FilteredBlock.FilteredTransactions {
		// 	// todo
		// 	buffer_end = append(buffer_end, fmt.Sprintf("end: %d %s %s", st, tx.GetTxid(), tx.TxValidationCode))
		// }
		// buffer_tot = append(buffer_tot, fmt.Sprintf("Time %8.2fs\tBlock %6d\tTx %6d\n", time.Since(now).Seconds(), fb.FilteredBlock.Number, len(fb.FilteredBlock.FilteredTransactions)))
	}
}
