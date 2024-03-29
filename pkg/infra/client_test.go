package infra_test

import (
	"github.com/Yunpeng-J/tape/pkg/infra"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"
)

var _ = Describe("Client", func() {

	Context("Client Error handling", func() {
		dummy := infra.Node{
			Addr: "invalid_addr",
		}
		logger := log.New()

		It("captures error from endorser", func() {
			_, err := infra.CreateEndorserClient(dummy, logger)
			Expect(err).Should(MatchError(ContainSubstring("error connecting to invalid_addr")))
		})
		It("captures error from broadcaster", func() {
			_, err := infra.CreateBroadcastClient(dummy, logger)
			Expect(err).Should(MatchError(ContainSubstring("error connecting to invalid_addr")))
		})
		It("captures error from DeliverFilter", func() {
			_, err := infra.CreateDeliverFilteredClient(dummy, logger)
			Expect(err).Should(MatchError(ContainSubstring("error connecting to invalid_addr")))
		})
	})

})
