package auction_test

import (
	"flag"
	"fmt"
	"github.com/cloudfoundry/yagnats"
	"github.com/onsi/auction/auctioneer"
	"github.com/onsi/auction/nats/repnatsclient"
	"github.com/onsi/auction/types"
	"github.com/onsi/auction/util"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
	"time"
)

var guids = []string{}
var natsAddrs = []string{}

var numReps int
var repResources int

var rules types.AuctionRules
var timeout time.Duration

var auctioneerMode string

// plumbing
var natsClient yagnats.NATSClient
var client types.TestRepPoolClient
var communicator types.AuctionCommunicator

func init() {
	flag.StringVar(&auctioneerMode, "auctioneerMode", "inprocess", "one of inprocess, remote")

	flag.IntVar(&(auctioneer.DefaultRules.MaxRounds), "maxRounds", auctioneer.DefaultRules.MaxRounds, "the maximum number of rounds per auction")
	flag.IntVar(&(auctioneer.DefaultRules.MaxBiddingPool), "maxBiddingPool", auctioneer.DefaultRules.MaxBiddingPool, "the maximum number of participants in the pool")
	flag.IntVar(&(auctioneer.DefaultRules.MaxConcurrent), "maxConcurrent", auctioneer.DefaultRules.MaxConcurrent, "the maximum number of concurrent auctions to run")
	flag.BoolVar(&(auctioneer.DefaultRules.RepickEveryRound), "repickEveryRound", auctioneer.DefaultRules.RepickEveryRound, "whether to repick every round")
}

func TestAuction(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Auction Suite")
}

var _ = BeforeSuite(func() {
	numReps = len(guids)
	repResources = 100

	fmt.Printf("Running in %s auctioneerMode\n", auctioneerMode)

	//parse flags to set up rules
	timeout = 500 * time.Millisecond

	rules = auctioneer.DefaultRules

	natsClient = yagnats.NewClient()
	clusterInfo := &yagnats.ConnectionCluster{}

	for _, addr := range natsAddrs {
		clusterInfo.Members = append(clusterInfo.Members, &yagnats.ConnectionInfo{
			Addr: addr,
		})
	}

	err := natsClient.Connect(clusterInfo)
	Ω(err).ShouldNot(HaveOccurred())

	client = repnatsclient.New(natsClient, timeout)

	if auctioneerMode == "inprocess" {
		communicator = func(auctionRequest types.AuctionRequest) types.AuctionResult {
			return auctioneer.Auction(client, auctionRequest)
		}
	} else if auctioneerMode == "remote" {
		communicator = func(auctionRequest types.AuctionRequest) types.AuctionResult {
			return auctioneer.RemoteAuction(natsClient, auctionRequest)
		}
	} else {
		panic("wat?")
	}
})

var _ = BeforeEach(func() {
	for _, guid := range guids {
		client.Reset(guid)
	}

	util.ResetGuids()
})
