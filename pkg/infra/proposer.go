package infra

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/hyperledger/fabric-protos-go/common"
	"github.com/hyperledger/fabric-protos-go/orderer"
	"github.com/hyperledger/fabric-protos-go/peer"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type Proposers struct {
	workers [][]*Proposer
	//one proposer per connection per peer
	client int
	logger *log.Logger
	token  chan struct{}
}

func CreateProposers(conn, client int, nodes []Node, burst int, logger *log.Logger) (*Proposers, error) {
	var ps [][]*Proposer
	var err error
	//one proposer per connection per peer
	token := make(chan struct{}, burst)
	for _, node := range nodes {
		row := make([]*Proposer, conn)
		for j := 0; j < conn; j++ {
			row[j], err = CreateProposer(node, logger, token)
			if err != nil {
				return nil, err
			}
		}
		ps = append(ps, row)
	}

	return &Proposers{workers: ps, client: client, logger: logger, token: token}, nil
}

func (ps *Proposers) Start(signed []chan *Elements, processed chan *Elements, done <-chan struct{}, config Config) {
	ps.logger.Infof("Start sending transactions.")
	interval := 1e9 / config.Rate * float64(len(config.Endorsers)/config.EndorserGroups)
	go func() {
		for {
			time.Sleep(time.Duration(interval) * time.Nanosecond)
			ps.token <- struct{}{}
		}
	}()
	expect := config.Rate / float64(config.NumOfConn*config.ClientPerConn*config.EndorserGroups)
	for i := 0; i < len(config.Endorsers); i++ {
		// peer connection should be config.ClientPerConn * config.NumOfConn
		for k := 0; k < config.ClientPerConn; k++ {
			for j := 0; j < config.NumOfConn; j++ {
				go ps.workers[i][j].Start(signed[i], processed, done, int(len(config.Endorsers)/config.EndorserGroups), k, j, expect)
			}
		}
	}
}

type Proposer struct {
	e      peer.EndorserClient
	Addr   string
	logger *log.Logger
	token  chan struct{}
}

func CreateProposer(node Node, logger *log.Logger, token chan struct{}) (*Proposer, error) {
	endorser, err := CreateEndorserClient(node, logger)
	if err != nil {
		return nil, err
	}

	return &Proposer{e: endorser, Addr: node.Addr, logger: logger, token: token}, nil
}

func (p *Proposer) getToken() {
	<-p.token
}

func (p *Proposer) Start(signed, processed chan *Elements, done <-chan struct{}, threshold int, ii int, jj int, expect float64) {
	cnt := 0
	var st int64
	base := time.Now().UnixNano()
	for {
		select {
		case s := <-signed:
			//send sign proposal to peer for endorsement
			p.getToken()
			st = time.Now().UnixNano()
			buffer_start <- fmt.Sprintf("start: %d %d_%s %d %d", st, global_txid2id[s.Txid], s.Txid, ii, jj)
			r, err := p.e.ProcessProposal(context.Background(), s.SignedProp)
			if err != nil || r.Response.Status < 200 || r.Response.Status >= 400 {
				if r == nil {
					p.logger.Errorf("Err processing proposal: %s, status: unknown, addr: %s \n", err, p.Addr)
				} else {
					p.logger.Errorf("Err processing proposal: %s, status: %d, message: %s, addr: %s \n", err, r.Response.Status, r.Response.Message, p.Addr)
				}
				continue
			}
			cnt += 1
			s.lock.Lock()
			//collect for endorsement
			s.Responses = append(s.Responses, r)
			if len(s.Responses) >= threshold {
				processed <- s
				// todo
				st := time.Now().UnixNano()
				buffer_proposal <- fmt.Sprintf("proposal: %d %d_%s", st, global_txid2id[s.Txid], s.Txid)
			}
			s.lock.Unlock()
		case <-done:
			return
		}
		if cnt >= 10 {
			tps := cnt * 1e9 / int(st-base)
			log.Println("endorser send rate", tps, "expect", expect)
			base = st
			cnt = 0
		}
	}
}

type Broadcasters struct {
	broadcasters []*Broadcaster
	token        chan struct{}
}

func CreateBroadcasters(orderer_client int, orderer Node, burst int, logger *log.Logger) (*Broadcasters, error) {
	bs := &Broadcasters{
		broadcasters: make([]*Broadcaster, orderer_client),
		token:        make(chan struct{}, burst),
	}
	for i := 0; i < orderer_client; i++ {
		broadcaster, err := CreateBroadcaster(orderer, bs.token, logger)
		if err != nil {
			return nil, err
		}
		bs.broadcasters[i] = broadcaster
	}

	return bs, nil
}

func (bs *Broadcasters) Start(envs <-chan *Elements, rate float64, errorCh chan error, done <-chan struct{}) {
	interval := 1e9 / rate
	go func() {
		for {
			bs.token <- struct{}{}
			time.Sleep(time.Duration(interval) * time.Nanosecond)
		}
	}()
	for _, b := range bs.broadcasters {
		go b.StartDraining(errorCh)
		go b.Start(envs, rate/float64(len(bs.broadcasters)), errorCh, done)
	}
}

type Broadcaster struct {
	c      orderer.AtomicBroadcast_BroadcastClient
	logger *log.Logger
	token  chan struct{}
}

func (b *Broadcaster) getToken() {
	<-b.token
}

func CreateBroadcaster(node Node, token chan struct{}, logger *log.Logger) (*Broadcaster, error) {
	client, err := CreateBroadcastClient(node, logger)
	if err != nil {
		return nil, err
	}

	return &Broadcaster{c: client, logger: logger, token: token}, nil
}

func (b *Broadcaster) Start(envs <-chan *Elements, rate float64, errorCh chan error, done <-chan struct{}) {
	b.logger.Debugf("Start sending broadcast")
	cnt := 0
	base := time.Now().UnixNano()
	var st int64
	for {
		select {
		case e := <-envs:
			// todo
			b.getToken()
			st = time.Now().UnixNano()
			buffer_sent <- fmt.Sprintf("sent: %d %d_%s", st, global_txid2id[e.Txid], e.Txid)
			err := b.c.Send(e.Envelope)
			if err != nil {
				errorCh <- err
			}
			cnt += 1

		case <-done:
			return
		}
		if cnt >= 20 {
			tps := cnt * 1e9 / int(st-base)
			log.Println("orderer send rate", tps, "expect", rate)
			cnt = 0
			base = st
		}
	}
}

func (b *Broadcaster) StartDraining(errorCh chan error) {
	for {
		res, err := b.c.Recv()
		if err != nil {
			if err == io.EOF {
				return
			}
			b.logger.Errorf("recv broadcast err: %+v, status: %+v\n", err, res)
			return
		}

		if res.Status != common.Status_SUCCESS {
			errorCh <- errors.Errorf("recv errouneous status %s", res.Status)
			return
		}
	}
}
