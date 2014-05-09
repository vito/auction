package auction_test

import (
	"flag"
	"fmt"
	"os/exec"

	"github.com/cloudfoundry/gunk/natsrunner"
	"github.com/onsi/auction/auctioneer"
	"github.com/onsi/auction/http/rephttpclient"
	"github.com/onsi/auction/lossyrep"
	"github.com/onsi/auction/nats/repnatsclient"
	"github.com/onsi/auction/representative"
	"github.com/onsi/auction/types"
	"github.com/onsi/auction/util"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"

	"testing"
	"time"
)

var repNodeBinary string

var mode string

var sessionsToTerminate []*gexec.Session

var natsPort int
var natsRunner *natsrunner.NATSRunner

var rules auctioneer.Rules
var timeout time.Duration

const InProcess = "inprocess"
const HTTP = "http"
const NATS = "nats"

var client types.TestRepPoolClient
var guids []string

var numReps = 50
var repResources = 100

func init() {
	flag.StringVar(&mode, "mode", "inprocess", "one of inprocess, http, nats")
}

func TestAuction(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Auction Suite")
}

var _ = BeforeSuite(func() {
	var err error

	fmt.Printf("Running in %s mode\n", mode)

	repNodeBinary, err = gexec.Build("github.com/onsi/auction/repnode")
	Ω(err).ShouldNot(HaveOccurred())

	//parse flags to set up rules
	timeout = 500 * time.Millisecond
	natsPort = 5222 + GinkgoParallelNode()

	natsRunner = natsrunner.NewNATSRunner(natsPort)

	rules = auctioneer.DefaultRules
	rules.MaxRounds = 100
	rules.RepickEveryRound = true

	sessionsToTerminate = []*gexec.Session{}

	natsRunner.Start()
	client, guids = buildClient(numReps, repResources)
})

var _ = BeforeEach(func() {
	for _, guid := range guids {
		client.Reset(guid)
	}

	util.ResetGuids()
})

var _ = AfterSuite(func() {
	for _, sess := range sessionsToTerminate {
		sess.Kill().Wait()
	}

	natsRunner.Stop()
})

func buildClient(numReps int, repResources int) (types.TestRepPoolClient, []string) {
	if mode == InProcess {
		lossyrep.LatencyMin = 2 * time.Millisecond
		lossyrep.LatencyMax = 12 * time.Millisecond
		lossyrep.Timeout = 50 * time.Millisecond
		lossyrep.Flakiness = 0.95

		guids := []string{}
		repMap := map[string]*representative.Representative{}

		for i := 0; i < numReps; i++ {
			guid := util.NewGuid("REP")
			guids = append(guids, guid)
			repMap[guid] = representative.New(guid, repResources)
		}

		client := lossyrep.New(repMap, map[string]bool{})
		return client, guids
	} else if mode == NATS {
		guids := []string{}

		for i := 0; i < numReps; i++ {
			guid := util.NewGuid("REP")

			serverCmd := exec.Command(
				repNodeBinary,
				"-guid", guid,
				"-natsAddr", fmt.Sprintf("127.0.0.1:%d", natsPort),
				"-resources", fmt.Sprintf("%d", repResources),
			)

			sess, err := gexec.Start(serverCmd, GinkgoWriter, GinkgoWriter)
			Ω(err).ShouldNot(HaveOccurred())
			Eventually(sess).Should(gbytes.Say("listening"))
			sessionsToTerminate = append(sessionsToTerminate, sess)

			guids = append(guids, guid)
		}

		client := repnatsclient.New(natsRunner.MessageBus, timeout)

		return client, guids
	} else if mode == HTTP {
		startPort := 18000 + (numReps * GinkgoParallelNode())
		guids := []string{}

		repMap := map[string]string{}

		for i := 0; i < numReps; i++ {
			guid := util.NewGuid("REP")
			port := startPort + i

			serverCmd := exec.Command(
				repNodeBinary,
				"-guid", guid,
				"-httpAddr", fmt.Sprintf("0.0.0.0:%d", port),
				"-resources", fmt.Sprintf("%d", repResources),
			)

			repMap[guid] = fmt.Sprintf("http://127.0.0.1:%d", port)

			sess, err := gexec.Start(serverCmd, GinkgoWriter, GinkgoWriter)
			Ω(err).ShouldNot(HaveOccurred())
			Eventually(sess).Should(gbytes.Say("serving"))
			sessionsToTerminate = append(sessionsToTerminate, sess)

			guids = append(guids, guid)
		}

		client := rephttpclient.New(repMap, timeout)

		return client, guids
	}

	panic("wat!")
}
