package smallbank

import (
	"bufio"
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"strconv"
	"sync"
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
	hotRate           float64
	transactionNumber int
	directory         string

	shardNumber     int
	shardNextClient map[int]int
	clientsPerShard int

	clients          int
	maxTxsPerSession int
	minTxsPerSession int
	maxHotPay        int

	accounts   []string
	hotAccount int

	zipf *utils.Zipf
	ch   chan string

	generators []*Generator
	resubmit   chan string
	mu         sync.Mutex
	cache      map[string][]string
	cacheShard map[string]int

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

func NewSmallBank(provider metrics.Provider, resub chan string) *SmallBank {
	res := &SmallBank{
		accountNumber:     viper.GetInt("accountNumber"),
		hotAccountRate:    viper.GetFloat64("hotAccountRate"),
		transactionNumber: viper.GetInt("transactionNumber"),
		directory:         viper.GetString("workloadDirectory"),
		clients:           viper.GetInt("clientsNumber"),
		clientsPerShard:   viper.GetInt("clientsPerEndorser"),
		maxTxsPerSession:  viper.GetInt("maxTxsPerSession"),
		minTxsPerSession:  viper.GetInt("minTxsPerSession"),
		maxHotPay:         viper.GetInt("maxHotPay"),
		hotRate:           viper.GetFloat64("hotRate"),
		accounts:          nil,
		ch:                make(chan string, 10000),
		resubmit:          resub,
		mu:                sync.Mutex{},
		cache:             make(map[string][]string),
		cacheShard:        make(map[string]int),
		metrics:           NewMetrics(provider),
		shardNumber:       viper.GetInt("shardNumber"),
		shardNextClient:   map[int]int{},
	}
	res.hotAccount = int(float64(res.accountNumber) * res.hotAccountRate)
	res.zipf = utils.NewZipf(res.accountNumber, viper.GetFloat64("zipfs"))
	for i := 0; i < res.shardNumber; i++ {
		res.shardNextClient[i] = 0
	}

	log.Printf("\n######\tworkload config (smallbank)\t######\n %v", res)

	if viper.GetString("transactionType") == "init" {
		go res.saveAccount()
		// os.RemoveAll(res.directory + "/account")
		// res.Init()
	} else if viper.GetString("transactionType") == "txn" {
		res.loadAccount()
		// res.GenerateTransaction()
	}
	return res
}

func (smallBank *SmallBank) ForEachClient(id int, session string) GeneratorT {
	gen := NewGenerator(smallBank, id, session)
	smallBank.generators = append(smallBank.generators, gen)
	return gen
}

func (smallBank *SmallBank) Resubmit() {
	for {
		txid := <-smallBank.resubmit
		smallBank.mu.Lock()
		shard := smallBank.cacheShard[txid]
		tx := smallBank.cache[txid]
		if len(tx) == 0 {
			log.Println("resub nil", txid)
			smallBank.mu.Unlock()
			continue
		}
		tx[0] = txid
		// log.Printf("shard resub txid %s shard %d", txid, shard)
		smallBank.mu.Unlock()
		smallBank.generators[shard].ch <- &tx
		// log.Println("resub true", txid)
	}
}

func (smallBank *SmallBank) Start() {
	go smallBank.Resubmit()
	chaincode := viper.GetString("smartContract")
	switch chaincode {
	case "smallbank":
		smallBank.generateSmallBank()
	case "KVStore":
		smallBank.generateKVStore()
	default:
		panic(fmt.Sprintf("unknown smart contract: %s", chaincode))
	}
}

func (smallBank *SmallBank) SampleAccount() int {
	tp := viper.GetString("sampleType")
	if tp == "zipf" {
		from := smallBank.zipf.Generate()
		id := from / smallBank.shardNumber
		rd := utils.RandomId(smallBank.shardNumber)
		return id*smallBank.shardNumber + rd
	} else if tp == "random" {
		var from int
		r := rand.Float64()
		if r < smallBank.hotRate {
			from = utils.RandomInRange(0, smallBank.hotAccount)
		} else {
			from = utils.RandomInRange(smallBank.hotAccount, smallBank.accountNumber)
		}
		return from
	} else {
		panic(fmt.Sprintf("unknown sample type: %s", tp))
	}
}

// round-robin
func (smallBank *SmallBank) getClientForShard(shard int) int {
	smallBank.mu.Lock()
	defer smallBank.mu.Unlock()
	client := smallBank.shardNextClient[shard]
	smallBank.shardNextClient[shard] += 1
	if smallBank.shardNextClient[shard] == smallBank.clientsPerShard {
		smallBank.shardNextClient[shard] = 0
	}
	client = shard*smallBank.clientsPerShard + client
	return client
}

func (smallBank *SmallBank) generateSmallBank() {
	// CreateAccount: srcId, srcName, amount, amount
	// DepositChecking: id+, amount
	// WriteCheck: id-, amount
	// TransactSavings: id+, amount
	// SendPayment: src-, dst+, amount
	// Amalgamate: src-, dst+
	// Query
	DepositChecking := viper.GetFloat64("DepositChecking")
	WriteCheck := viper.GetFloat64("WriteCheck") + DepositChecking
	TransactSavings := viper.GetFloat64("TransactSavings") + WriteCheck
	SendPayment := viper.GetFloat64("SendPayment") + TransactSavings
	Amalgamate := viper.GetFloat64("Amalgamate") + SendPayment
	wg := sync.WaitGroup{}
	wkt := viper.GetInt("WorkloadThread")
	wg.Add(wkt)
	txnPerThread := int(smallBank.transactionNumber / wkt)
	for j := 0; j < wkt; j++ {
		// clients
		go func() {

			for i := 0; i < txnPerThread; i++ {
				smallBank.metrics.CreateCounter.With("generator", "0").Add(1)
				from := smallBank.SampleAccount()
				shard := from % smallBank.shardNumber
				// log.Printf("shard: %d, len %d account %d", shard, len(smallBank.generators), from)
				txid := utils.GetName(40)
				txid += "_" + smallBank.accounts[from]
				// if _, ok := smallBank.cache[txid]; ok {
				// 	panic(fmt.Sprintf("txid %s exist", txid))
				// }

				var tx []string
				txType := rand.Float64()
				if txType < DepositChecking {
					tx = []string{txid, "DepositChecking", smallBank.accounts[from], "1"}

				} else if txType < WriteCheck {
					tx = []string{txid, "WriteCheck", smallBank.accounts[from], "1"}

				} else if txType < TransactSavings {
					tx = []string{txid, "TransactSavings", smallBank.accounts[from], "1"}

				} else if txType < SendPayment {
					to := from
					for to == from {
						to = smallBank.SampleAccount()
					}
					txid += "2" + smallBank.accounts[to]
					tx = []string{txid, "SendPayment", smallBank.accounts[from], smallBank.accounts[to], "1"}
				} else if txType < Amalgamate {
					to := from
					for to == from {
						to = smallBank.SampleAccount()
					}
					txid += "2" + smallBank.accounts[to]
					tx = []string{txid, "Amalgamate", smallBank.accounts[from], smallBank.accounts[to]}
				} else {
					panic("the sum of transaction ratio should be 1")
				}
				client := smallBank.getClientForShard(shard)
				smallBank.generators[client].ch <- &tx
				smallBank.mu.Lock()
				smallBank.cacheShard[tx[0]] = client
				smallBank.cache[tx[0]] = tx
				smallBank.mu.Unlock()
			}
			wg.Done()
		}()
	}
	wg.Wait()
	val, _ := os.LookupEnv("RESUBMIT")
	resubmitFlag := val == "true"
	if resubmitFlag == false {
		for i := 0; i < smallBank.clients; i++ {
			smallBank.generators[i].ch <- &[]string{} // stop signal
		}
	}
}

func (smallBank *SmallBank) generateKVStore() {
	// KV2, KV4, KV8, KV16
	KV2 := viper.GetFloat64("KV2")
	KV4 := viper.GetFloat64("KV4") + KV2
	KV8 := viper.GetFloat64("KV8") + KV4
	KV16 := viper.GetFloat64("KV16") + KV8
	for i := 0; i < smallBank.transactionNumber; i++ {
		txid := utils.GetName(20)
		tx := []string{txid}
		shard := -1
		txType := rand.Float64()
		if txType < KV2 {
			tx = append(tx, "KV2")
			acts := map[int]int{}
			for len(acts) < 2 {
				acid := smallBank.SampleAccount()
				if _, ok := acts[acid]; !ok {
					acts[acid] = 1
				} else {
					acts[acid] += 1
				}
			}
			for k, _ := range acts {
				tx = append(tx, smallBank.accounts[k])
				shard = k % smallBank.clients
			}
		} else if txType < KV4 {
			tx = append(tx, "KV4")
			acts := map[int]int{}
			for len(acts) < 4 {
				acid := smallBank.SampleAccount()
				if _, ok := acts[acid]; !ok {
					acts[acid] = 1
				} else {
					acts[acid] += 1
				}
			}
			for k, _ := range acts {
				tx = append(tx, smallBank.accounts[k])
				shard = k % smallBank.clients
			}

		} else if txType < KV8 {
			tx = append(tx, "KV8")
			acts := map[int]int{}
			for len(acts) < 8 {
				acid := smallBank.SampleAccount()
				if _, ok := acts[acid]; !ok {
					acts[acid] = 1
				} else {
					acts[acid] += 1
				}
			}
			for k, _ := range acts {
				tx = append(tx, smallBank.accounts[k])
				shard = k % smallBank.clients
			}

		} else if txType < KV16 {
			tx = append(tx, "KV16")
			acts := map[int]int{}
			for len(acts) < 16 {
				acid := smallBank.SampleAccount()
				if _, ok := acts[acid]; !ok {
					acts[acid] = 1
				} else {
					acts[acid] += 1
				}
			}
			for k, _ := range acts {
				tx = append(tx, smallBank.accounts[k])
				shard = k % smallBank.clients
			}

		} else {
			panic("the sum of transaction ratio should be 1")
		}
		client := smallBank.getClientForShard(shard)
		smallBank.generators[client].ch <- &tx
	}

	for i := 0; i < smallBank.clients; i++ {
		smallBank.generators[i].ch <- &[]string{} // stop signal
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
