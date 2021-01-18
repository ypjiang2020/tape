package infra

import (
	"strconv"
	"fmt"
	"os"
	"math/rand"
	"time"
	"bufio"
	"strings"
)


var chs = []rune("qwertyuiopasdfghjklzxcvbnmQWERTYUIOPASDFGHJKLZXCVBNM1234567890!@#$%^&*()=")
var accounts_file string = "ACCOUNTS"
var transactions_file string = "TRANSACTIONS"
var accounts []string


func getName(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = chs[rand.Intn(len(chs))]
	}
	return string(b)
}

func generate() []string {
	transactions_type := "transfer_money"
	if _, err := os.Stat(accounts_file); os.IsNotExist(err) {
		transactions_type = "create_accounts"
	}

	var res []string
	// fmt.Println(transactions_type)

	if transactions_type == "transfer_money" {
		src := rand.Intn(len(accounts))
		dst := rand.Intn(len(accounts)) // maybe same
		// TODO: support other transactions 
		// (e.g., Amalgamate, TransactionsSavings, WriteCheck, DepositChecking)
		res = append(res, "SendPayment") 
		res = append(res, accounts[src])
		res = append(res, accounts[dst])
		res = append(res, "1")
	} else {
	   	idx := getName(64)
		res = append(res, "CreateAccount")
		res = append(res, idx)
		res = append(res, idx)
		res = append(res, strconv.Itoa(1e9))
		res = append(res, strconv.Itoa(1e9))
	}
	return res
}

func GenerateWorkload(n int) [][]string {
	rand.Seed(time.Now().UnixNano())
	// rand.Seed(667)

	if _, err := os.Stat(accounts_file); os.IsNotExist(err) {
		fmt.Printf("create %d accounts\n", n)
	} else {
		fmt.Printf("transfer money: %d\n", n)

		f, _ := os.Open(accounts_file)
		input := bufio.NewScanner(f) 
		for input.Scan() {
			accounts = append(accounts, input.Text())
		}
		fmt.Printf("read %d accounts from %s\n", len(accounts), accounts_file)
	}

	i := 0
	var res [][]string
	for i = 0; i < n; i++ {
		res = append(res, generate())
	}
	if _, err := os.Stat(accounts_file); err == nil {
		// save transactions to file for checking
		os.Remove(transactions_file)
		f, err := os.Create(transactions_file) 
		if err != nil {
			fmt.Println("create file failed", err)
		}
		defer f.Close()

		for i = 0; i < n; i++ {
			f.WriteString(strconv.Itoa(i) + " " + strings.Join(res[i], " ") )
			f.WriteString("\n")
		}

	} else {
		// save accounts to file 
		f, err := os.Create(accounts_file) 
		if err != nil {
			fmt.Println("create file failed", err)
		}
		defer f.Close()

		for i = 0; i < n; i++ {
			f.WriteString(res[i][1] + "\n")
		}
	}

	return res
}
