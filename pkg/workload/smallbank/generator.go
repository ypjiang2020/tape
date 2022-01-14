package smallbank

import (
	"fmt"
	"math/rand"
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

	session         string
	seq             int
	clientsPerShard int
	crdtOnly        bool
}

func NewGenerator(smlbk *SmallBank, id_ int, session_ string) *Generator {
	res := &Generator{
		smallBank:       smlbk,
		id:              strconv.Itoa(id_),
		ch:              make(chan *[]string, viper.GetInt("generatorBuffer")),
		session:         session_,
		seq:             id_ % viper.GetInt("clientsPerEndorser"),
		clientsPerShard: viper.GetInt("clientsPerEndorser"),
		crdtOnly:        viper.GetBool("crdtOnly"),
	}
	res.start()
	return res
}

func (gen *Generator) Generate() []string {
	if gen.crdtOnly {
		gen.session = utils.GetName(20)
		gen.seq = 0
	}
	gen.smallBank.metrics.GeneratorCounter.With("generator", gen.id).Add(1)
	temp := *(<-gen.ch)
	if len(temp) > 0 {
		temp[0] = fmt.Sprintf("%d_+=+_%s_+=+_%s", gen.seq, gen.session, temp[0])
	}
	gen.seq += gen.clientsPerShard
	// log.Println(temp)
	return temp
}

func (gen *Generator) Stop() []string {
	session := utils.GetName(20)
	txid := fmt.Sprintf("%d_+=+_%s_+=+_%s#end#", 0, session, utils.GetName(20))
	idx := utils.GetName(64) + "#end#"
	return []string{txid, "CreateAccount", idx, idx, "1", "1"}
}

func (gen *Generator) start() {
	if viper.GetString("transactionType") == "init" {
		go gen.createAccount()
	} else {
		// go gen.sendPayment()
	}
}

func (gen *Generator) createAccount() {
	// session := utils.GetName(20)
	money := strconv.Itoa(1e9)
	txsPerClient := gen.smallBank.accountNumber/gen.smallBank.clients + 1
	for j := 0; j < txsPerClient; j++ {
		gen.smallBank.metrics.CreateCounter.With("generator", gen.id).Add(1)
		// txid := fmt.Sprintf("%d_+=+_%s_+=+_%s", j, session, utils.GetName(20))
		idx := utils.GetName(64)
		// log.Printf("client %s key %s", gen.id, idx)
		gen.smallBank.ch <- idx
		gen.ch <- &[]string{idx, "CreateAccount", idx, idx, money, money}
	}
	gen.ch <- &[]string{} // stop signal
}

func (gen *Generator) sendPayment() {
	txsPerClient := gen.smallBank.transactionNumber/gen.smallBank.clients + 1
	done := 0
	genid, _ := strconv.Atoi(gen.id) // utils.RandomInRange(0, gen.smallBank.hotAccount) // hot buyer account
	// workload type: hot buyer, hot seller, normal
	pHotBuyer := viper.GetFloat64("hotBuyer")
	pHotSeller := viper.GetFloat64("hotSeller") + pHotBuyer
	for done < txsPerClient {
		session := utils.GetName(20)
		p := rand.Float64()
		if p < pHotBuyer {
			// hot buyer
			k := 0
			txThisSession := utils.RandomInRange(gen.smallBank.minTxsPerSession, gen.smallBank.maxTxsPerSession)
			from := utils.RandomInRange(0, gen.smallBank.hotAccount)/gen.smallBank.clients*gen.smallBank.clients + genid // hot buyer account
			for from >= gen.smallBank.hotAccount {
				from = utils.RandomInRange(0, gen.smallBank.hotAccount)/gen.smallBank.clients*gen.smallBank.clients + genid // hot buyer account
			}
			for k < txThisSession && done < txsPerClient {
				txid := fmt.Sprintf("%d_+=+_%s_+=+_%s", k, session, utils.GetName(20))
				k += 1
				done += 1
				to := utils.RandomInRange(0, gen.smallBank.accountNumber) // all accounts
				for from == to {
					to = utils.RandomInRange(0, gen.smallBank.accountNumber) // all accounts
				}
				gen.smallBank.metrics.CreateCounter.With("generator", gen.id).Add(1)
				gen.ch <- &[]string{txid, "SendPayment", gen.smallBank.accounts[from], gen.smallBank.accounts[to], "1"}
			}
		} else if p < pHotSeller {
			// hot seller
			to := utils.RandomInRange(gen.smallBank.hotAccount, gen.smallBank.hotAccount*2) // hot seller account
			txid := fmt.Sprintf("%d_+=+_%s_+=+_%s", 0, session, utils.GetName(20))
			done += 1
			from := utils.RandomInRange(0, gen.smallBank.accountNumber)/gen.smallBank.clients*gen.smallBank.clients + genid // normal accounts
			for from >= gen.smallBank.accountNumber {
				from = utils.RandomInRange(0, gen.smallBank.accountNumber)/gen.smallBank.clients*gen.smallBank.clients + genid // normal accounts
			}
			gen.smallBank.metrics.CreateCounter.With("generator", gen.id).Add(1)
			gen.ch <- &[]string{txid, "SendPayment", gen.smallBank.accounts[from], gen.smallBank.accounts[to], "1"}
		} else {
			// normal
			txid := fmt.Sprintf("%d_+=+_%s_+=+_%s", 0, session, utils.GetName(20))
			done += 1
			from := utils.RandomInRange(0, gen.smallBank.accountNumber)/gen.smallBank.clients*gen.smallBank.clients + genid // normal accounts
			for from >= gen.smallBank.accountNumber {
				from = utils.RandomInRange(0, gen.smallBank.accountNumber)/gen.smallBank.clients*gen.smallBank.clients + genid // normal accounts
			}
			to := utils.RandomInRange(0, gen.smallBank.accountNumber) // normal accounts
			for to == from {
				to = utils.RandomInRange(0, gen.smallBank.accountNumber) // normal accounts
			}
			gen.smallBank.metrics.CreateCounter.With("generator", gen.id).Add(1)
			gen.ch <- &[]string{txid, "SendPayment", gen.smallBank.accounts[from], gen.smallBank.accounts[to], "1"}
		}
	}
	gen.ch <- &[]string{} // stop signal
}
