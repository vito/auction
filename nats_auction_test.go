package auction_test

import (
	"fmt"
	"os/exec"
	"time"

	"github.com/cloudfoundry/gunk/natsrunner"
	"github.com/onsi/auction/auctioneer"
	"github.com/onsi/auction/instance"
	"github.com/onsi/auction/repnatsclient"
	"github.com/onsi/auction/util"
	"github.com/onsi/auction/visualization"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Auctioneering via NATS", func() {
	var repResources int
	var rules auctioneer.Rules

	var numServers int
	var repGuids []string
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

		repGuids = make([]string, numServers)
		sessions = make([]*gexec.Session, numServers)

		for i := 0; i < numServers; i++ {
			guid := fmt.Sprintf("NATS-%d", i)

			serverCmd := exec.Command(
				serverBin,
				"-guid", guid,
				"-natsAddr", fmt.Sprintf("127.0.0.1:%d", natsPort),
				"-resources", fmt.Sprintf("%d", repResources),
			)

			sess, err := gexec.Start(serverCmd, GinkgoWriter, GinkgoWriter)
			Ω(err).ShouldNot(HaveOccurred())

			Eventually(sess).Should(gbytes.Say("listening"))

			repGuids[i] = guid
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

			client := repnatsclient.New(natsRunner.MessageBus, timeout)

			results, duration := auctioneer.HoldAuctionsFor(client, instances, repGuids, rules)

			visualization.PrintReport(client, results, repGuids, duration)
		})
	})

	Context("when starting from empty", func() {
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
			client := repnatsclient.New(natsRunner.MessageBus, timeout)

			instances := []instance.Instance{}
			for color, num := range initialInstances {
				for i := 0; i < num; i++ {
					instances = append(instances, instance.New(color, 1))
				}
			}

			results, duration := auctioneer.HoldAuctionsFor(client, instances, repGuids[:40], rules)
			visualization.PrintReport(client, results, repGuids, duration)

			instances = []instance.Instance{}
			for color, num := range newInstances {
				for i := 0; i < num; i++ {
					instances = append(instances, instance.New(color, 1))
				}
			}

			results, duration = auctioneer.HoldAuctionsFor(client, instances, repGuids, rules)

			visualization.PrintReport(client, results, repGuids, duration)
		})
	})
})
