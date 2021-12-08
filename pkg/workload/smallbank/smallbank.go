package smallbank

import (
	"bufio"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"time"

	"github.com/spf13/viper"

	"github.com/Yunpeng-J/tape/pkg/metrics"
	"github.com/Yunpeng-J/tape/pkg/workload/utils"
)

// seq + session + txid

// params:
// clients number: session number

type SmallBank struct {
	accountNumber     int
	hotAccountRate    float64
	transactionNumber int
	directory         string

	clients          int
	maxTxsPerSession int
	minTxsPerSession int
	maxHotPay        int

	accounts   []string
	hotAccount int

	zipf *utils.Zipf
	ch   chan string

	metrics *Metrics
}

func init() {
	viper.SetDefault("accountNumber", 10000)
	viper.SetDefault("hotAccountRate", 0.01)
	viper.SetDefault("transactionNumber", 10000)
	viper.SetDefault("workloadDirectory", "./__workload")
	viper.SetDefault("maxTxsPerSession", 30)
	viper.SetDefault("minTxsPerSession", 2)
	viper.SetDefault("maxHotPay", 10)
	viper.SetDefault("zipfs", 1.0)
	viper.SetDefault("clientsPerEndorser", 1)
}

func NewSmallBank(provider metrics.Provider) *SmallBank {
	res := &SmallBank{
		accountNumber:     viper.GetInt("accountNumber"),
		hotAccountRate:    viper.GetFloat64("hotAccountRate"),
		transactionNumber: viper.GetInt("transactionNumber"),
		directory:         viper.GetString("workloadDirectory"),
		clients:           viper.GetInt("clientsNumber"),
		maxTxsPerSession:  viper.GetInt("maxTxsPerSession"),
		minTxsPerSession:  viper.GetInt("minTxsPerSession"),
		maxHotPay:         viper.GetInt("maxHotPay"),
		accounts:          nil,
		ch:                make(chan string, 10000),
		metrics:           NewMetrics(provider),
	}
	res.hotAccount = int(float64(res.accountNumber) * res.hotAccountRate)
	res.zipf = utils.NewZipf(res.accountNumber, viper.GetFloat64("zipfs"))

	log.Printf("\n######\tworkload config (smallbank)\t######\n %v", res)

	if viper.GetString("transactionType") == "init" {
		go res.saveAccount()
		// os.RemoveAll(res.directory + "/account")
		// res.Init()
	} else {
		res.loadAccount()
		// res.GenerateTransaction()
	}
	return res
}

func (smallBank *SmallBank) ForEachClient(id int) GeneratorT {
	return NewGenerator(smallBank, id)
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
				from := smallBank.zipf.Generate()
				to := smallBank.zipf.Generate()
				for to == from {
					to = smallBank.zipf.Generate()
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

func (smallBank *SmallBank) loadAccount() {
	accountFile := smallBank.directory + "/account/account.txt"
	f, err := os.Open(accountFile)
	if err != nil {
		panic(fmt.Sprintf("cannot open: %s", accountFile))
	}
	input := bufio.NewScanner(f)
	for input.Scan() {
		smallBank.accounts = append(smallBank.accounts, input.Text())
	}
	log.Printf("load %d accounts from %s", len(smallBank.accounts), accountFile)
}

func (smallBank *SmallBank) saveAccount() {
	err := os.MkdirAll(smallBank.directory+"/account", os.ModePerm)
	if err != nil {
		panic(fmt.Sprintf("create directory failed: %v", err))
	}
	f, err := os.Create(smallBank.directory + "/account/account.txt")
	if err != nil {
		panic(fmt.Sprintf("create file account.txt failed: %v", err))
	}
	defer f.Close()
	i := 0
	for i < smallBank.accountNumber {
		account := <-smallBank.ch
		if account[len(account)-5:] != "#end#" {
			f.WriteString(account + "\n")
			i += 1
		}
	}
}
