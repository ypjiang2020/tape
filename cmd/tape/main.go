package main

import (
	"fmt"
	"os"

	"tape/pkg/infra"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"
)

const (
	loglevel = "TAPE_LOGLEVEL"
)

var (
	app = kingpin.New("tape", "A performance test tool for Hyperledger Fabric")

	run = app.Command("run", "Start the tape program").Default()
	con = run.Flag("config", "Path to config file").Required().Short('c').String()

	seed  = run.Flag("seed", "seed").Default("0").Int()
	rate  = run.Flag("rate", "[Optional] Creates tx rate, default 0 as unlimited").Default("0").Float64()
	burst = run.Flag("burst", "[Optional] Burst size for Tape, should bigger than rate").Default("1000").Int()
	e2e   = run.Flag("e2e", "end to end").Default("true").Bool()

	num_of_conn     = run.Flag("num_of_conn", "number of connections").Default("16").Int()
	client_per_conn = run.Flag("client_per_conn", "clients per connection").Default("16").Int()
	orderer_client  = run.Flag("orderer_client", "orderer clients").Default("20").Int()
	threads         = run.Flag("thread", "signature thread").Default("20").Int()
	endorser_groups = run.Flag("endorser_group", "endorser groups").Required().Int()

	num_of_transactions  = run.Flag("number", "Number of tx for shot").Default("50000").Short('n').Int()
	time_of_transactions = run.Flag("time", "time of tx for shot (default 120s)").Default("120").Short('t').Int()
	tx_type              = run.Flag("txtype", "transaction type [put, conflict]").Required().String()

	version = app.Command("version", "Show version information")
)

func loadConfig() infra.Config {
	config, err := infra.LoadConfig(*con)
	if err != nil {
		log.Fatalf("load config error: %v\n", err)
	}
	config.Seed = *seed
	config.Rate = *rate
	config.Burst = *burst
	config.End2end = *e2e
	config.ClientPerConn = *client_per_conn
	config.NumOfConn = *num_of_conn
	config.OrdererClients = *orderer_client
	config.Threads = *threads
	config.EndorserGroups = *endorser_groups
	config.NumOfTransactions = *num_of_transactions
	config.TimeOfTransactions = *time_of_transactions
	config.TxType = *tx_type

	return config
}

func main() {
	var err error

	logger := log.New()
	logger.SetLevel(log.WarnLevel)
	if customerLevel, customerSet := os.LookupEnv(loglevel); customerSet {
		if lvl, err := log.ParseLevel(customerLevel); err == nil {
			logger.SetLevel(lvl)
		}
	}

	fullCmd := kingpin.MustParse(app.Parse(os.Args[1:]))
	switch fullCmd {
	case version.FullCommand():
		fmt.Printf(infra.GetVersionInfo())
	case run.FullCommand():
		checkArgs(rate, burst, logger)
		config := loadConfig()
		err = infra.Process(config, logger)
	default:
		err = errors.Errorf("invalid command: %s", fullCmd)
	}

	if err != nil {
		logger.Error(err)
		os.Exit(1)
	}
	os.Exit(0)
}

func checkArgs(rate *float64, burst *int, logger *log.Logger) {
	if *rate < 0 {
		os.Stderr.WriteString("tape: error: rate must be zero (unlimited) or positive number\n")
		os.Exit(1)
	}
	if *burst < 1 {
		os.Stderr.WriteString("tape: error: burst at least 1\n")
		os.Exit(1)
	}

	if int64(*rate) > int64(*burst) {
		fmt.Printf("As rate %d is bigger than burst %d, real rate is burst\n", int64(*rate), int64(*burst))
	}

	logger.Infof("Will use rate %f as send rate\n", *rate)
	logger.Infof("Will use %d as burst\n", burst)
}
