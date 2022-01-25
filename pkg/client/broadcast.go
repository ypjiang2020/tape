package client

import (
	"fmt"
	"io"

	"github.com/Yunpeng-J/fabric-protos-go/common"
	"github.com/Yunpeng-J/fabric-protos-go/orderer"
	log "github.com/sirupsen/logrus"
)

type Broadcaster struct {
	id                 int
	orderer            orderer.AtomicBroadcast_BroadcastClient
	client2broadcaster chan *common.Envelope
	done               chan struct{}
}

func NewBroadcaster(id int, orderer Node, c2b chan *common.Envelope) *Broadcaster {
	bc := &Broadcaster{client2broadcaster: c2b, id: id, done: make(chan struct{})}

	var err error
	bc.orderer, err = CreateBroadcastClient(orderer, logger)
	if err != nil {
		panic(fmt.Sprintf("create ordererClient failed: %v", err))
	}
	return bc
}

func (bc *Broadcaster) Run(done chan struct{}) {
	go bc.StartDraining(done)
	defer func() {
		close(bc.done)
	}()
	for {
		select {
		case env := <-bc.client2broadcaster:
			// log.Printf("orderer id %d receive", bc.id)
			err := bc.orderer.Send(env)
			if err != nil {
				log.Errorf("send to order failed: %v", err)
				return
			}
		case <-done:
			return
		}
	}
}
func (bc *Broadcaster) StartDraining(done chan struct{}) {
	// TODO: metric
	for {
		select {
		case <-done:
			return
		case <-bc.done:
			return
		default:
			res, err := bc.orderer.Recv()
			if err != nil {
				if err == io.EOF {
					return
				}
				log.Printf("recv from orderer error: %v", err)
				return
			}
			if res.Status != common.Status_SUCCESS {
				log.Printf("recv from orderer, status: %s", res.Status)
				return
			}
		}
	}
}
