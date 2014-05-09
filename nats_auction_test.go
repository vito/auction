package auction_test

import (
	"fmt"
	"os/exec"
	"time"

	"github.com/cloudfoundry/gunk/natsrunner"
	"github.com/onsi/auction/auctioneer"
	"github.com/onsi/auction/instance"
	"github.com/onsi/auction/natsauctioneer"
	"github.com/onsi/auction/repnatsclient"
	"github.com/onsi/auction/representative"
	"github.com/onsi/auction/util"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Auctioneering via NATS", func() {
	var repResources int
	var rules auctioneer.Rules

	var numServers int
	var servers []string
	var sessions []*gexec.Session

	var natsPort int
	var natsRunner *natsrunner.NATSRunner
	var timeout time.Duration

	BeforeEach(func() {
		timeout = 500 * time.Millisecond

		natsPort = 5222 + GinkgoParallelNode()
		natsRunner = natsrunner.NewNATSRunner(natsPort)
		natsRunner.Start()

		numServers = 100

		repResources = 100
		util.ResetGuids()
		rules = auctioneer.DefaultRules
		rules.MaxRounds = 100
	})

	JustBeforeEach(func() {
		serverBin, err := gexec.Build("github.com/onsi/auction/repnatsserver")
		Ω(err).ShouldNot(HaveOccurred())

		servers = make([]string, numServers)
		sessions = make([]*gexec.Session, numServers)

		for i := 0; i < numServers; i++ {
			guid := fmt.Sprintf("server%d", i)

			serverCmd := exec.Command(
				serverBin,
				"-guid", guid,
				"-natsAddr", fmt.Sprintf("127.0.0.1:%d", natsPort),
			)

			sess, err := gexec.Start(serverCmd, GinkgoWriter, GinkgoWriter)
			Ω(err).ShouldNot(HaveOccurred())

			Eventually(sess).Should(gbytes.Say("listening"))

			servers[i] = guid
			sessions[i] = sess
		}
	})

	AfterEach(func() {
		for _, sess := range sessions {
			sess.Kill().Wait()
		}

		natsRunner.Stop()
	})

	Context("with empty representatives and single-instance apps", func() {
		var numApps int

		BeforeEach(func() {
			numApps = 400
		})

		It("should distribute evenly", func() {
			instances := []instance.Instance{}
			for i := 0; i < numApps; i++ {
				instances = append(instances, instance.New(util.NewGuid("APP"), 1))
			}

			representatives := make([]representative.Rep, numServers)
			for i, guid := range servers {
				representatives[i] = repnatsclient.New(natsRunner.MessageBus, guid, timeout)
			}

			results := natsauctioneer.HoldAuctionsFor(natsRunner.MessageBus, instances, representatives, rules)

			printReport(results, representatives, rules, false)
		})
	})

	FContext("when starting from empty", func() {
		var newInstances, initialInstances map[string]int

		BeforeEach(func() {

			initialInstances = map[string]int{
				"green":  700,
				"red":    400,
				"cyan":   100,
				"yellow": 250,
				"gray":   100,
			}

			newInstances = map[string]int{
				"green":  1000,
				"red":    750,
				"cyan":   500,
				"yellow": 250,
				"gray":   100,
			}
		})

		It("should distribute evenly", func() {
			representatives := make([]representative.Rep, numServers)
			for i, guid := range servers {
				representatives[i] = repnatsclient.New(natsRunner.MessageBus, guid, timeout)
			}

			instances := []instance.Instance{}
			for color, num := range initialInstances {
				for i := 0; i < num; i++ {
					instances = append(instances, instance.New(color, 1))
				}
			}

			results := natsauctioneer.HoldAuctionsFor(natsRunner.MessageBus, instances, representatives[:40], rules)
			printReport(results, representatives, rules, true)

			instances = []instance.Instance{}
			for color, num := range newInstances {
				for i := 0; i < num; i++ {
					instances = append(instances, instance.New(color, 1))
				}
			}

			results = natsauctioneer.HoldAuctionsFor(natsRunner.MessageBus, instances, representatives, rules)

			printReport(results, representatives, rules, true)
		})
	})
})
