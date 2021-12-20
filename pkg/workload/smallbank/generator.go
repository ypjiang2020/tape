package smallbank

import (
	"fmt"
	"log"
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
	temp := *(<-gen.ch)
	log.Println(temp)
	return temp
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
	txsPerClient := gen.smallBank.transactionNumber/gen.smallBank.clients + 1
	done := 0
	for done < txsPerClient {
		session := utils.GetName(20)
		txThisSession := gen.smallBank.minTxsPerSession //utils.RandomInRange(gen.smallBank.minTxsPerSession, gen.smallBank.maxTxsPerSession)

		// workload type: hot buyer, hot seller, normal
		pHotBuyer := viper.GetFloat64("hotBuyer")
		pHotSeller := viper.GetFloat64("hotSeller") + pHotBuyer
		p := rand.Float64()
		if p < pHotBuyer {
			// hot buyer
			k := 0
			from := utils.RandomInRange(0, gen.smallBank.hotAccount) // hot buyer account
			for k < txThisSession && done < txsPerClient {
				txid := fmt.Sprintf("%d_+=+_%s_+=+_%s", k, session, utils.GetName(20))
				k += 1
				done += 1
				to := utils.RandomInRange(gen.smallBank.hotAccount*2, gen.smallBank.accountNumber) // normal accounts
				for to == from {
					to = utils.RandomId(gen.smallBank.accountNumber)
				}
				gen.smallBank.metrics.CreateCounter.With("generator", gen.id).Add(1)
				gen.ch <- &[]string{txid, "SendPayment", gen.smallBank.accounts[from], gen.smallBank.accounts[to], "1"}
			}
		} else if p < pHotSeller {
			// hot seller
			k := 0
			to := utils.RandomInRange(gen.smallBank.hotAccount, gen.smallBank.hotAccount*2) // hot seller account
			for k < txThisSession && done < txsPerClient {
				txid := fmt.Sprintf("%d_+=+_%s_+=+_%s", k, session, utils.GetName(20))
				k += 1
				done += 1
				from := utils.RandomInRange(gen.smallBank.hotAccount*2, gen.smallBank.accountNumber) // normal accounts
				for to == from {
					from = utils.RandomInRange(gen.smallBank.hotAccount*2, gen.smallBank.accountNumber)
				}
				gen.smallBank.metrics.CreateCounter.With("generator", gen.id).Add(1)
				gen.ch <- &[]string{txid, "SendPayment", gen.smallBank.accounts[from], gen.smallBank.accounts[to], "1"}
			}
		} else {
			// normal
			k := 0
			for k < txThisSession && done < txsPerClient {
				txid := fmt.Sprintf("%d_+=+_%s_+=+_%s", k, session, utils.GetName(20))
				k += 1
				done += 1
				from := utils.RandomInRange(gen.smallBank.hotAccount*2, gen.smallBank.accountNumber) // normal accounts
				to := utils.RandomInRange(gen.smallBank.hotAccount*2, gen.smallBank.accountNumber)   // normal accounts
				for to == from {
					from = utils.RandomInRange(gen.smallBank.hotAccount*2, gen.smallBank.accountNumber)
				}
				gen.smallBank.metrics.CreateCounter.With("generator", gen.id).Add(1)
				gen.ch <- &[]string{txid, "SendPayment", gen.smallBank.accounts[from], gen.smallBank.accounts[to], "1"}
			}
		}
	}
	gen.ch <- &[]string{} // stop signal
}
