package infra

import (
	"strconv"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

var global_txid2id map[string]int

func StartCreateProposal(config Config, crypto *Crypto, raw chan *Elements, errorCh chan error, logger *log.Logger) {
	global_txid2id = make(map[string]int)
	wg := newWorkloadGenerator(config)
	chaincodeCtorJSONs := wg.GenerateWorkload()
	for i := 0; i < config.NumOfTransactions; i++ {
		chaincodeCtorJSON := chaincodeCtorJSONs[i]
		temptxid := strconv.Itoa(i) + "_" + getName(20)
		if config.Check_Txid {
			temptxid = ""
		}
		prop, txid, err := CreateProposal(
			temptxid,
			crypto,
			config.Channel,
			config.Chaincode,
			config.Version,
			chaincodeCtorJSON,
		)
		if err != nil {
			errorCh <- errors.Wrapf(err, "error creating proposal")
			return
		}
		global_txid2id[txid] = i
		raw <- &Elements{Proposal: prop, Txid: txid}
	}
}
