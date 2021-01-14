package infra

import (
	"context"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
)

func StartCreateProposal(num int, burst int, r float64, config Config, crypto *Crypto, raw chan *Elements, errorCh chan error, logger *log.Logger) {
	limit := rate.Inf
	ctx := context.Background()
	if r > 0 {
		limit = rate.Limit(r)
	}
	limiter := rate.NewLimiter(limit, burst)
	chaincodeCtorJSONs := GenerateWorkload(num)
	// fmt.Println(config.Channel)
	// fmt.Println(config.Chaincode)

	for i := 0; i < num; i++ {
		chaincodeCtorJSON := chaincodeCtorJSONs[i]
		// fmt.Println(chaincodeCtorJSON)
		prop, err := CreateProposal(
			crypto,
			config.Channel,
			config.Chaincode,
			config.Version,
			chaincodeCtorJSON,
		)
		if err != nil {
			errorCh <- errors.Wrapf(err, "error creating proposal")
			return
		}

		if err = limiter.Wait(ctx); err != nil {
			errorCh <- errors.Wrapf(err, "error creating proposal")
			return
		}

		raw <- &Elements{Proposal: prop}
	}
}
