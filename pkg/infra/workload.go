package infra

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

var chs = []rune("qwertyuiopasdfghjklzxcvbnmQWERTYUIOPASDFGHJKLZXCVBNM1234567890!@#$%^&*()=")
var accounts_file string = "ACCOUNTS"
var transactions_file string = "TRANSACTIONS"

func getName(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = chs[rand.Intn(len(chs))]
	}
	return string(b)
}

func randomId(n int) int {
	res := rand.Intn(n)
	return res
}

type WorkloadGenerator struct {
	config   Config
	accounts []string
}

func newWorkloadGenerator(config Config) *WorkloadGenerator {
	return &WorkloadGenerator{config: config}
}

func (wg *WorkloadGenerator) generate() []string {
	var res []string
	// fmt.Println(transactions_type)

	if wg.config.TxType == "conflict" {
		src := rand.Intn(len(wg.accounts))
		dst := rand.Intn(len(wg.accounts)) // maybe same
		// TODO: support other transactions
		// (e.g., Amalgamate, TransactionsSavings, WriteCheck, DepositChecking)
		res = append(res, "SendPayment")
		res = append(res, wg.accounts[src])
		res = append(res, wg.accounts[dst])
		res = append(res, "1")
	} else if wg.config.TxType == "put" {
		idx := getName(64)
		res = append(res, "CreateAccount")
		res = append(res, idx)
		res = append(res, idx)
		res = append(res, strconv.Itoa(1e9))
		res = append(res, strconv.Itoa(1e9))
	}
	return res
}

func (wg *WorkloadGenerator) GenerateWorkload() [][]string {
	if wg.config.Seed == 0 {
		rand.Seed(time.Now().UnixNano())
	} else {
		rand.Seed(int64(wg.config.Seed))
	}
	if wg.config.TxType == "put" {
		fmt.Printf("create %d accounts\n", wg.config.NumOfTransactions)
	} else if wg.config.TxType == "conflict" {
		if _, err := os.Stat(accounts_file); os.IsNotExist(err) {
			log.Fatalf("please create accounts first: %v\n", err)
		}

		fmt.Printf("transfer money: %d\n", wg.config.NumOfTransactions)
		f, _ := os.Open(accounts_file)
		input := bufio.NewScanner(f)
		for input.Scan() {
			wg.accounts = append(wg.accounts, input.Text())
		}
		fmt.Printf("read %d accounts from %s\n", len(wg.accounts), accounts_file)
	}

	i := 0
	var res [][]string
	for i = 0; i < wg.config.NumOfTransactions; i++ {
		res = append(res, wg.generate())
	}
	os.Remove(transactions_file)
	f, err := os.Create(transactions_file)
	if err != nil {
		fmt.Println("create file failed", err)
	}
	defer f.Close()

	for i = 0; i < wg.config.NumOfTransactions; i++ {
		f.WriteString(strconv.Itoa(i) + " " + strings.Join(res[i], " ") + "\n")
	}
	if wg.config.TxType == "put" {
		// save accounts to file
		f, err := os.Create(accounts_file)
		if err != nil {
			fmt.Println("create file failed", err)
		}
		defer f.Close()

		for i = 0; i < wg.config.NumOfTransactions; i++ {
			f.WriteString(res[i][1] + "\n")
		}
	}
	return res
}
