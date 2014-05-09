package auction_test

import (
	"github.com/onsi/auction/lossyrep"
	"github.com/onsi/auction/representative"
	"github.com/onsi/auction/types"
	"github.com/onsi/auction/util"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	"testing"
)

var repNodeBinary string

var mode string

const InProcess = "inprocess"
const HTTP = "http"
const NATS = "nats"

func TestAuction(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Auction Suite")
}

var _ = BeforeSuite(func() {
	var err error
	repNodeBinary, err = gexec.Build("github.com/onsi/auction/repnode")
	Ω(err).ShouldNot(HaveOccurred())

	mode = InProcess
	// mode = HTTP
	// mode = NATS

	//parse flags to set up rules
})

func buildClient(numReps int, repResources int) (types.TestRepPoolClient, []string) {
	if mode == InProcess {
		guids := []string{}
		repMap := map[string]*representative.Representative{}

		for i := 0; i < numReps; i++ {
			guid := util.NewGuid("REP")
			guids = append(guids, guid)
			repMap[guid] = representative.New(guid, repResources)
		}

		client := lossyrep.New(repMap, map[string]bool{})
		return client, guids
	}

	panic("wat!")
}
