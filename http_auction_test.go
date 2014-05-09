package auction_test

// import (
// 	"fmt"
// 	"os/exec"

// 	. "github.com/onsi/ginkgo"
// 	. "github.com/onsi/gomega"
// 	"github.com/onsi/gomega/gbytes"
// 	"github.com/onsi/gomega/gexec"

// 	"github.com/onsi/auction/auctioneer"
// 	"github.com/onsi/auction/instance"
// 	"github.com/onsi/auction/repclient"
// 	"github.com/onsi/auction/util"
// 	"github.com/onsi/auction/visualization"
// )

// var _ = Describe("Auctioneering via HTTP", func() {
// 	var repResources int
// 	var rules auctioneer.Rules

// 	var numServers int
// 	var servers map[string]string
// 	var sessions []*gexec.Session
// 	var repGuids []string

// 	BeforeEach(func() {
// 		numServers = 10
// 		repResources = 100

// 		util.ResetGuids()
// 		rules = auctioneer.DefaultRules
// 		rules.MaxRounds = 100
// 	})

// 	JustBeforeEach(func() {
// 		serverBin, err := gexec.Build("github.com/onsi/auction/repserver")
// 		Ω(err).ShouldNot(HaveOccurred())

// 		startPort := 18000 + (numServers * GinkgoParallelNode())

// 		servers = map[string]string{}
// 		sessions = make([]*gexec.Session, numServers)
// 		repGuids = make([]string, numServers)

// 		for i := 0; i < numServers; i++ {
// 			port := startPort + i
// 			guid := fmt.Sprintf("HTTP-%d", i)
// 			serverCmd := exec.Command(
// 				serverBin,
// 				"-guid", guid,
// 				"-listenAddr", fmt.Sprintf("0.0.0.0:%d", port),
// 				"-resources", fmt.Sprintf("%d", repResources),
// 			)

// 			sess, err := gexec.Start(serverCmd, GinkgoWriter, GinkgoWriter)
// 			Ω(err).ShouldNot(HaveOccurred())

// 			Eventually(sess).Should(gbytes.Say("serving on"))

// 			servers[guid] = fmt.Sprintf("http://127.0.0.1:%d", port)
// 			sessions[i] = sess
// 			repGuids[i] = guid
// 		}
// 	})

// 	AfterEach(func() {
// 		for _, sess := range sessions {
// 			sess.Kill().Wait()
// 		}
// 	})

// 	Context("with empty representatives and single-instance apps", func() {
// 		var numApps int

// 		BeforeEach(func() {
// 			numApps = 500
// 		})

// 		It("should distribute evenly", func() {
// 			instances := []instance.Instance{}
// 			for i := 0; i < numApps; i++ {
// 				instances = append(instances, instance.New(util.NewGuid("APP"), 1))
// 			}

// 			client := repclient.New(servers)

// 			results, duration := auctioneer.HoldAuctionsFor(client, instances, repGuids, rules)

// 			visualization.PrintReport(client, results, repGuids, duration, rules)
// 		})
// 	})

// 	// PContext("when starting from empty", func() {
// 	// 	var newInstances map[string]int
// 	// 	var repDistributions []int

// 	// 	BeforeEach(func() {
// 	// 		newInstances = map[string]int{
// 	// 			"green":  100,
// 	// 			"red":    75,
// 	// 			"cyan":   50,
// 	// 			"yellow": 25,
// 	// 			"gray":   10,
// 	// 		}
// 	// 	})

// 	// 	It("should distribute evenly", func() {
// 	// 		instances := []instance.Instance{}
// 	// 		for color, num := range newInstances {
// 	// 			for i := 0; i < num; i++ {
// 	// 				instances = append(instances, instance.New(color, 1))
// 	// 			}
// 	// 		}

// 	// 		representatives := []representative.Rep{}
// 	// 		for _, repoApps := range repDistributions {
// 	// 			numExistingApps := repoApps
// 	// 			flaky := false
// 	// 			if repoApps < 0 {
// 	// 				numExistingApps = -repoApps
// 	// 				flaky = true
// 	// 			}
// 	// 			existingInstances := map[string]instance.Instance{}
// 	// 			for i := 0; i < numExistingApps; i++ {
// 	// 				inst := instance.New(util.RandomFrom("green", "red", "yellow", "cyan", "gray"), 1)
// 	// 				existingInstances[inst.InstanceGuid] = inst
// 	// 			}
// 	// 			representatives = append(representatives, lossyrep.New(repResources, flaky, existingInstances))
// 	// 		}

// 	// 		results := auctioneer.HoldAuctionsFor(instances, representatives, rules)

// 	// 		printReport(results, representatives, rules, true)
// 	// 	})
// 	// })
// })
