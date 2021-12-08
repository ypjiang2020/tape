package smallbank

import (
	"fmt"
	"strconv"

	"github.com/Yunpeng-J/tape/pkg/workload/utils"
	"github.com/spf13/viper"
)

type GeneratorT interface {
	Generate() []string
	Stop() []string
}

type Generator struct {
	smallBank *SmallBank
	id        string
	ch        chan *[]string
}

func NewGenerator(smlbk *SmallBank, id int) *Generator {
	res := &Generator{
		smallBank: smlbk,
		id:        strconv.Itoa(id),
		ch:        make(chan *[]string, viper.GetInt("generatorBuffer")),
	}
	res.start()
	return res
}

func (gen *Generator) Generate() []string {
	gen.smallBank.metrics.GeneratorCounter.With("generator", gen.id).Add(1)
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
	txsPerClient := gen.smallBank.accountNumber/gen.smallBank.clients + 1
	for j := 0; j < txsPerClient; j++ {
		gen.smallBank.metrics.CreateCounter.With("generator", gen.id).Add(1)
		txid := fmt.Sprintf("%d_+=+_%s_+=+_%s", j, session, utils.GetName(20))
		idx := utils.GetName(64)
		gen.smallBank.ch <- idx
		gen.ch <- &[]string{txid, "CreateAccount", idx, idx, money, money}
	}
	gen.ch <- &[]string{} // stop signal
}

func (gen *Generator) sendPayment() {
	clients := viper.GetInt("accountNumber")
	for {
		// loop for timeout
		session := utils.GetName(20)
		// number of transactions in this session
		txThisSession := utils.RandomInRange(gen.smallBank.minTxsPerSession, gen.smallBank.maxTxsPerSession)
		k := 0
		for k < txThisSession {
			k += 1
			txid := fmt.Sprintf("%d_+=+_%s_+=+_%s", k, session, utils.GetName(20))
			from := utils.RandomId(clients) // gen.smallBank.zipf.Generate()
			to := utils.RandomId(clients)   //gen.smallBank.zipf.Generate()
			for to == from {
				to = utils.RandomId(clients) //gen.smallBank.zipf.Generate()
			}
			gen.smallBank.metrics.CreateCounter.With("generator", gen.id).Add(1)
			gen.ch <- &[]string{txid, "SendPayment", gen.smallBank.accounts[from], gen.smallBank.accounts[to], "1"}
		}
	}
}
