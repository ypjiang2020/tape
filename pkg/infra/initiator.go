package infra

import (
	"strconv"

	"github.com/Yunpeng-J/fabric-protos-go/peer"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func StartCreateProposal(config Config, crypto *Crypto, raw chan *Elements, errorCh chan error, logger *log.Logger) {
	wg := newWorkloadGenerator(config)
	chaincodeCtorJSONs := wg.GenerateWorkload()
	props := make([]*peer.Proposal, config.NumOfTransactions)
	txids := make([]string, config.NumOfTransactions)
	session := getName(20)
	for i := 0; i < config.NumOfTransactions; i++ {
		chaincodeCtorJSON := chaincodeCtorJSONs[i]
		temptxid := strconv.Itoa(i) + "_+=+_" + session + "_+=+_" + getName(20)
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
		props[i] = prop
		txids[i] = txid
	}
	go func() {
		for i := 0; i < len(props); i++ {
			raw <- &Elements{Proposal: props[i], Txid: txids[i]}
		}
	}()
}
