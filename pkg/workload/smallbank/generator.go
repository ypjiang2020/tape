package smallbank

import (
	"fmt"
	"github.com/Yunpeng-J/tape/pkg/workload/utils"
	"github.com/spf13/viper"
	"strconv"
)

type Generator struct {
	smallBank *SmallBank
	id        int
	ch        chan *[]string
}

func NewGenerator(smlbk *SmallBank, id int) *Generator {
	res := &Generator{
		smallBank: smlbk,
		id:        id,
		ch: make(chan *[]string, 10),
	}
	res.start()
	return res
}

func (gen *Generator) Generate() []string {
	return *(<-gen.ch)
}

func (gen *Generator) Stop() []string {
	session := utils.GetName(20)
	txid := fmt.Sprintf("%d_+=+_%s_+=+_%s#end#", 0, session, utils.GetName(20))
	idx := utils.GetName(64)
	return []string{txid, "CreateAccount", idx, idx, "1", "1"}
}

func (gen *Generator) start() {
	if viper.GetString("transactionType") == "init" {
		go gen.createAccount()
	} else {
		go gen.sendPayment()
	}
}

func (gen *Generator) createAccount() {
	session := utils.GetName(20)
	money := strconv.Itoa(1e9)
	txsPerClient := gen.smallBank.accountNumber / gen.smallBank.clients + 1
	for j := 0; j < txsPerClient; j++ {
		txid := fmt.Sprintf("%d_+=+_%s_+=+_%s", j, session, utils.GetName(20))
		idx := utils.GetName(64)
		gen.smallBank.ch <- idx
		gen.ch	<- &[]string{txid, "CreateAccount", idx, idx, money, money}
	}
	gen.ch <- &[]string{} // stop signal
}

func (gen *Generator) sendPayment() {
	for {
		// loop for timeout
		session := utils.GetName(20)
		// number of transactions in this session
		txThisSession := utils.RandomInRange(gen.smallBank.minTxsPerSession, gen.smallBank.maxTxsPerSession)
		k := 0
		for k < txThisSession {
			k += 1
			txid := fmt.Sprintf("%d_+=+_%s_+=+_%s", k, session, utils.GetName(20))
			from := gen.smallBank.zipf.Generate()
			to := gen.smallBank.zipf.Generate()
			for to == from {
				to = gen.smallBank.zipf.Generate()
			}
			gen.ch <- &[]string{txid, "SendPayment", gen.smallBank.accounts[from], gen.smallBank.accounts[to], "1"}
		}
	}
}
