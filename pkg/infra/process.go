package infra

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/hyperledger/fabric-protos-go/common"
	log "github.com/sirupsen/logrus"
)

var endorsement_file = "ENDORSEMENT"
var global_txid2id map[string]int

var (
	MAX_BUF         = 100010
	buffer_start    = make(chan string, MAX_BUF) // start: timestamp txid clientid connectionid
	buffer_proposal = make(chan string, MAX_BUF) // proposal: timestamp txid
	buffer_sent     = make(chan string, MAX_BUF) // sent: timestamp txid
	buffer_end      = make(chan string, MAX_BUF) // end: timestamp txid [VALID/MVCC]
	buffer_tot      = make(chan string, MAX_BUF) // null
	g_wg            = sync.WaitGroup{}
)

func print_benchmark(logdir string, done <-chan struct{}) {
	g_wg.Add(1)
	defer g_wg.Done()
	f, err := os.Create(logdir)
	if err != nil {
		log.Fatalf("open log %s failed: %v\n", logdir, err)
	}
	defer f.Close()
	for {
		select {
		case res := <-buffer_start:
			f.WriteString(res + "\n")
		case res := <-buffer_proposal:
			f.WriteString(res + "\n")
		case res := <-buffer_sent:
			f.WriteString(res + "\n")
		case res := <-buffer_end:
			f.WriteString(res + "\n")
		case <-done:
			for len(buffer_start) > 0 {
				f.WriteString(<-buffer_start + "\n")
			}
			for len(buffer_proposal) > 0 {
				f.WriteString(<-buffer_proposal + "\n")
			}
			for len(buffer_sent) > 0 {
				f.WriteString(<-buffer_sent + "\n")
			}
			for len(buffer_end) > 0 {
				f.WriteString(<-buffer_end + "\n")
			}
			for len(buffer_tot) > 0 {
				f.WriteString(<-buffer_tot + "\n")
			}
			return
		}
	}
}

func e2e(config Config, logger *log.Logger) error {
	crypto, err := config.LoadCrypto()
	if err != nil {
		return err
	}
	raw := make(chan *Elements, config.Burst)
	signed := make([]chan *Elements, len(config.Endorsers))
	processed := make(chan *Elements, config.Burst)
	envs := make(chan *Elements, config.Burst)
	done := make(chan struct{})
	finishCh := make(chan struct{})
	errorCh := make(chan error, config.Burst)
	assember := &Assembler{Signer: crypto, EndorserGroups: config.EndorserGroups, Conf: config}
	go print_benchmark(config.Logdir, done)

	for i := 0; i < len(config.Endorsers); i++ {
		signed[i] = make(chan *Elements, config.Burst)
	}

	for i := 0; i < config.Threads; i++ {
		go assember.StartIntegrator(processed, envs, errorCh, done)
	}

	proposor, err := CreateProposers(config.NumOfConn, config.ClientPerConn, config.Endorsers, config.Burst, logger)
	if err != nil {
		return err
	}

	broadcaster, err := CreateBroadcasters(config.OrdererClients, config.Orderer, config.Burst, logger)
	if err != nil {
		return err
	}

	observer, err := CreateObserver(config.Channel, config.Committer, crypto, logger)
	if err != nil {
		return err
	}
	go StartCreateProposal(config, crypto, raw, errorCh, logger)
	time.Sleep(10 * time.Second)
	start := time.Now()
	for i := 0; i < config.Threads; i++ {
		go assember.StartSigner(raw, signed, errorCh, done)
	}
	proposor.Start(signed, processed, done, config)
	broadcaster.Start(envs, config.Rate, errorCh, done)
	go observer.Start(int32(config.NumOfTransactions), errorCh, finishCh, start, &assember.Abort)

	for {
		select {
		case err = <-errorCh:
			return err
		case <-finishCh:
			duration := time.Since(start)
			logger.Infof("Completed processing transactions.")
			buffer_tot <- fmt.Sprintf("tx: %d, duration: %+v, tps: %f", config.NumOfTransactions, duration, float64(config.NumOfTransactions)/duration.Seconds())
			buffer_tot <- fmt.Sprintf("abort rate because of the different ledger height: %d %.2f%%", assember.Abort, float64(assember.Abort)/float64(config.NumOfTransactions)*100)
			close(done)
			g_wg.Wait()
			return nil
		}
	}
}

func breakdown_phase1(config Config, logger *log.Logger) error {
	crypto, err := config.LoadCrypto()
	if err != nil {
		return err
	}
	raw := make(chan *Elements, config.Burst)
	signed := make([]chan *Elements, len(config.Endorsers))
	processed := make(chan *Elements, config.Burst)
	done := make(chan struct{})
	errorCh := make(chan error, config.Burst)
	assember := &Assembler{Signer: crypto, EndorserGroups: config.EndorserGroups, Conf: config}
	go print_benchmark(config.Logdir, done)

	for i := 0; i < len(config.Endorsers); i++ {
		signed[i] = make(chan *Elements, config.Burst)
	}

	proposor, err := CreateProposers(config.NumOfConn, config.ClientPerConn, config.Endorsers, config.Burst, logger)
	if err != nil {
		return err
	}

	go StartCreateProposal(config, crypto, raw, errorCh, logger)
	time.Sleep(10 * time.Second)
	start := time.Now()

	for i := 0; i < config.Threads; i++ {
		go assember.StartSigner(raw, signed, errorCh, done)
	}
	proposor.Start(signed, processed, done, config)

	// phase1: send proposals to endorsers
	var cnt int32 = 0
	var buffer [][]byte
	var txids []string
	for i := 0; i < config.NumOfTransactions; i++ {
		select {
		case err = <-errorCh:
			return err
		case tx := <-processed:
			res, err := assember.Assemble(tx)
			if err != nil {
				fmt.Println("error: assemble endorsement to envelop")
				return err
			}
			bytes, err := json.Marshal(res.Envelope)
			if err != nil {
				fmt.Println("error: marshal envelop")
				return err
			}
			cnt += 1
			buffer = append(buffer, bytes)
			txids = append(txids, tx.Txid)
			if cnt+assember.Abort >= int32(config.NumOfTransactions) {
				break
			}
		}
	}
	duration := time.Since(start)

	logger.Infof("Completed endorsing transactions.")
	buffer_tot <- fmt.Sprintf("tx: %d, duration: %+v, tps: %f", config.NumOfTransactions, duration, float64(config.NumOfTransactions)/duration.Seconds())
	buffer_tot <- fmt.Sprintf("abort rate because of the different ledger height: %d %.2f%%", assember.Abort, float64(assember.Abort)/float64(config.NumOfTransactions)*100)
	close(done)
	g_wg.Wait()
	// persistency
	mfile, _ := os.Create(endorsement_file)
	defer mfile.Close()
	mw := bufio.NewWriter(mfile)
	for i := range buffer {
		mw.Write(buffer[i])
		mw.WriteByte('\n')
		mw.WriteString(txids[i])
		mw.WriteByte('\n')
	}
	mw.Flush()
	return nil
}
func breakdown_phase2(config Config, logger *log.Logger) error {
	crypto, err := config.LoadCrypto()
	if err != nil {
		return err
	}
	envs := make(chan *Elements, config.Burst)
	done := make(chan struct{})
	finishCh := make(chan struct{})
	errorCh := make(chan error, config.Burst)
	go print_benchmark(config.Logdir, done)

	broadcaster, err := CreateBroadcasters(config.OrdererClients, config.Orderer, config.Burst, logger)
	if err != nil {
		return err
	}

	observer, err := CreateObserver(config.Channel, config.Committer, crypto, logger)
	if err != nil {
		return err
	}

	mfile, _ := os.Open(endorsement_file)
	defer mfile.Close()
	mscanner := bufio.NewScanner(mfile)
	var txids []string
	TXs := make([]common.Envelope, config.NumOfTransactions)
	i := 0
	for mscanner.Scan() {
		bytes := mscanner.Bytes()
		json.Unmarshal(bytes, &TXs[i])
		if mscanner.Scan() {
			txid := mscanner.Text()
			txids = append(txids, txid)
		}
		i++
	}
	start := time.Now()
	broadcaster.Start(envs, config.Rate, errorCh, done)
	var temp0 int32 = 0
	go observer.Start(int32(config.NumOfTransactions), errorCh, finishCh, start, &temp0)
	go func() {
		for i := 0; i < config.NumOfTransactions; i++ {
			var item Elements
			item.Envelope = &TXs[i]
			item.Txid = txids[i]
			global_txid2id[item.Txid] = i
			envs <- &item
		}
	}()

	for {
		select {
		case err = <-errorCh:
			return err
		case <-finishCh:
			duration := time.Since(start)
			logger.Infof("Completed processing transactions.")
			buffer_tot <- fmt.Sprintf("tx: %d, duration: %+v, tps: %f", config.NumOfTransactions, duration, float64(config.NumOfTransactions)/duration.Seconds())
			close(done)
			g_wg.Wait()
			return nil
		}
	}

}

func Process(config Config, logger *log.Logger) error {
	global_txid2id = make(map[string]int)
	if config.End2end {
		fmt.Println("e2e")
		return e2e(config, logger)
	} else {
		if _, err := os.Stat(endorsement_file); err == nil {
			fmt.Println("phase2")
			// phase2: broadcast transactions to order
			return breakdown_phase2(config, logger)
		} else {
			fmt.Println("phase1")
			// phase1: send proposals to endorsers {
			return breakdown_phase1(config, logger)
		}

	}
}
