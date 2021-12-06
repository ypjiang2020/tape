package smallbank

import (
	"fmt"
	"github.com/spf13/viper"
	"log"
	"math"
	"os"
	"strconv"
	"time"

	"github.com/Yunpeng-J/tape/pkg/workload/utils"
)

// seq + session + txid

// params:
// clients number: session number
//

type SmallBank struct {
	accountNumber     int
	hotAccountRate    float64
	zipfs             float64
	transactionNumber int
	directory         string

	clients          int
	maxTxsPerSession int
	minTxsPerSession int
	maxHotPay        int

	accounts   []string
	hotAccount int
}

var defaultSmallbank = &SmallBank{
	accountNumber:     100000,
	hotAccountRate:    0.01,
	zipfs:             1,
	transactionNumber: 100000,
	directory:         ".",
	clients:           80,
	maxTxsPerSession:  50,
	minTxsPerSession:  5,
	maxHotPay:         20,
	accounts:          nil,
	hotAccount:        1000,
}

func NewSmallBank(actNum int, clientNum int, hotRate, zipfs float64, dir string) *SmallBank {
	return &SmallBank{
		accountNumber:  actNum,
		hotAccountRate: hotRate,
		zipfs:          zipfs,
		transactionNumber: 100000,
		directory:      dir,
		clients:        clientNum,
		maxTxsPerSession: 50,
		minTxsPerSession: 5,
		maxHotPay: 20,
		accounts: nil,
		hotAccount:     int(float64(actNum) * hotRate),
	}
}

func (smallBank *SmallBank) Init() {
	now := time.Now()
	defer func() {
		log.Printf("smallbank init in %d ms\n", time.Since(now).Milliseconds())
	}()
	err := os.MkdirAll(smallBank.directory+"/account", os.ModePerm)
	if err != nil {
		panic(fmt.Sprintf("create directory failed: %v", err))
	}
	txsPerClient := smallBank.accountNumber / smallBank.clients
	leaves := smallBank.accountNumber % smallBank.clients
	if leaves != 0 {
		txsPerClient += 1
	}

	accounts, err := os.Create(smallBank.directory + "/account/account.txt")
	if err != nil {
		panic(fmt.Sprintf("create file account.txt failed: %v", err))
	}
	defer accounts.Close()
	for i := 0; i < smallBank.clients; i++ {
		f, err := os.Create(smallBank.directory + "/account/" + strconv.Itoa(i) + "_account.txt")
		if err != nil {
			panic(fmt.Sprintf("create file %d_account.txt failed: %v", i, err))
		}
		session := utils.GetName(20)
		for j := 0; j < txsPerClient; j++ {
			txid := fmt.Sprintf("%d_+=+_%s_+=+_%s", j, session, utils.GetName(20))
			idx := utils.GetName(64)
			smallBank.accounts = append(smallBank.accounts, idx)
			accounts.WriteString(idx + "\n")
			f.WriteString(txid + " " + idx + "\n")
		}
		f.Close()
	}
}

func (smallBank *SmallBank) GenerateTransaction() {
	zipf := utils.NewZipf(smallBank.accountNumber, smallBank.zipfs)
	now := time.Now()
	defer func() {
		log.Printf("smallbank generate transaction in %d ms\n", time.Since(now).Milliseconds())
	}()
	err := os.MkdirAll(smallBank.directory+"/transaction", os.ModePerm)
	if err != nil {
		panic(fmt.Sprintf("create directory failed: %v", err))
	}
	txsPerClient := smallBank.transactionNumber / smallBank.clients
	leaves := smallBank.transactionNumber % smallBank.clients
	if leaves != 0 {
		txsPerClient += 1
	}

	for i := 0; i < smallBank.clients; i++ {
		f, err := os.Create(smallBank.directory + "/transaction/" + strconv.Itoa(i) + "_transaction.txt")
		if err != nil {
			panic(fmt.Sprintf("create file %d_transaction.txt failed: %v", i, err))
		}
		j := 0
		for j < txsPerClient {
			session := utils.GetName(20)
			// number of transactions in this session
			txThisSession := int(math.Min(float64(txsPerClient-j), float64(utils.RandomInRange(smallBank.minTxsPerSession, smallBank.maxTxsPerSession))))
			k := 0
			for k < txThisSession {
				k += 1
				txid := fmt.Sprintf("%d_+=+_%s_+=+_%s", k, session, utils.GetName(20))
				from := zipf.Generate()
				to := zipf.Generate()
				for to == from {
					to = zipf.Generate()
				}
				f.WriteString(fmt.Sprintf("%s %s %s\n", txid, smallBank.accounts[from], smallBank.accounts[to]))
			}
			j += txThisSession
		}
		f.Close()
	}
}

func (smallbank *SmallBank) createAccount(txid string) []string {
	idx := utils.GetName(64)
	return []string{
		txid,              // transaction ID
		"CreateAccount",   // function name
		idx,               // account id
		idx,               // account name
		strconv.Itoa(1e9), // saving balance
		strconv.Itoa(1e9), // checking balance
	}
}

func (smallbank *SmallBank) sendPayment(txid string, src, dst int) []string {
	return []string{
		txid,                    // transaction ID
		"SendPayment",           // function name
		smallbank.accounts[src], // source account
		smallbank.accounts[dst], // dest account
		"1",                     // money
	}
}

func (smallBank *SmallBank) LoadAccount() {
	// TODO

}

func (smallBank *SmallBank) Workload() []string {
	// TODO
	if viper.GetString("transactionType") == "init" {
		return smallBank.createAccount(utils.GetName(20))
	} else {
		return smallBank.sendPayment("", 0, 0)
	}

}
func (smallBank *SmallBank) Stop() []string {
	return smallBank.createAccount(utils.GetName(20) + "#end#")
}
