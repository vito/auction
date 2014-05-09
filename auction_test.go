package auction_test

import (
	"time"

	"github.com/onsi/auction/auctioneer"
	"github.com/onsi/auction/instance"
	"github.com/onsi/auction/lossyrep"
	"github.com/onsi/auction/representative"
	"github.com/onsi/auction/util"
	"github.com/onsi/auction/visualization"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Ω

var _ = Describe("Auction", func() {
	var repResources int
	var rules auctioneer.Rules

	BeforeEach(func() {
		lossyrep.LatencyMin = 2 * time.Millisecond
		lossyrep.LatencyMax = 12 * time.Millisecond
		lossyrep.Timeout = 50 * time.Millisecond
		lossyrep.Flakiness = 0.95

		repResources = 100
		util.ResetGuids()
		rules = auctioneer.DefaultRules
		rules.MaxRounds = 100
	})

	Context("with empty representatives and single-instance apps", func() {
		var numApps int
		var numReps int

		BeforeEach(func() {
			numApps = 500
			numReps = 10
		})

		It("should distribute evenly", func() {
			instances := []instance.Instance{}
			for i := 0; i < numApps; i++ {
				instances = append(instances, instance.New(util.NewGuid("APP"), 1))
			}

			var client *lossyrep.LossyRep
			repGuids := []string{}
			repMap := map[string]*representative.Representative{}

			for i := 0; i < numReps; i++ {
				guid := util.NewGuid("REP")
				repGuids = append(repGuids, guid)
				repMap[guid] = representative.New(guid, repResources, nil)
			}

			client = lossyrep.New(repMap, map[string]bool{})

			results, duration := auctioneer.HoldAuctionsFor(client, instances, repGuids, rules)

			visualization.PrintReport(client, results, repGuids, duration)
		})
	})

	Context("with non-empty representatives (and single-instance apps)", func() {
		var numApps int
		var repDistributions []int

		BeforeEach(func() {
			numApps = 100
			repDistributions = []int{100, 20, 10, -7, 19, 32, -42, 71, 10, 20, 13, 82, 36, 42, 16, 13, 28, 57, 12, -2}
		})

		It("should distribute evenly", func() {
			instances := []instance.Instance{}
			for i := 0; i < numApps; i++ {
				instances = append(instances, instance.New(util.NewGuid("APP"), 1))
			}

			var client *lossyrep.LossyRep
			repGuids := []string{}
			repMap := map[string]*representative.Representative{}
			flakyMap := map[string]bool{}

			for _, repoApps := range repDistributions {
				guid := util.NewGuid("REP")
				numExistingApps := repoApps
				if repoApps < 0 {
					numExistingApps = -repoApps
					flakyMap[guid] = true
				}
				existingInstances := map[string]instance.Instance{}
				for i := 0; i < numExistingApps; i++ {
					inst := instance.New(util.NewGuid("APP"), 1)
					existingInstances[inst.InstanceGuid] = inst
				}
				repGuids = append(repGuids, guid)
				repMap[guid] = representative.New(guid, repResources, existingInstances)
			}

			client = lossyrep.New(repMap, flakyMap)

			results, duration := auctioneer.HoldAuctionsFor(client, instances, repGuids, rules)

			visualization.PrintReport(client, results, repGuids, duration)
		})
	})

	Context("apps with multiple instances", func() {
		var newInstances map[string]int
		var repDistributions []int

		generateNewInstances := func(newInstances map[string]int) []instance.Instance {
			instances := []instance.Instance{}
			for color, num := range newInstances {
				for i := 0; i < num; i++ {
					instances = append(instances, instance.New(color, 1))
				}
			}
			return instances
		}

		generateReps := func(repDistributions []int) ([]string, *lossyrep.LossyRep) {
			var client *lossyrep.LossyRep
			repGuids := []string{}
			repMap := map[string]*representative.Representative{}
			flakyMap := map[string]bool{}

			for _, repoApps := range repDistributions {
				guid := util.NewGuid("REP")
				numExistingApps := repoApps
				if repoApps < 0 {
					numExistingApps = -repoApps
					flakyMap[guid] = true
				}
				existingInstances := map[string]instance.Instance{}
				for i := 0; i < numExistingApps; i++ {
					inst := instance.New(util.RandomFrom("green", "red", "yellow", "cyan", "gray"), 1)
					existingInstances[inst.InstanceGuid] = inst
				}
				repGuids = append(repGuids, guid)
				repMap[guid] = representative.New(guid, repResources, existingInstances)
			}

			client = lossyrep.New(repMap, flakyMap)

			return repGuids, client
		}

		Context("when starting from a (terrible) initial distribution", func() {
			BeforeEach(func() {
				newInstances = map[string]int{
					"green":  30,
					"red":    27,
					"cyan":   10,
					"yellow": 22,
					"gray":   8,
				}
				repDistributions = []int{100, 20, 10, -7, 19, 32, -42, 71, 10, 20, 13, 82, 36, 42, 16, 13, 28, 57, 12, -2}
			})

			It("should distribute evenly", func() {
				instances := generateNewInstances(newInstances)
				repGuids, client := generateReps(repDistributions)
				results, duration := auctioneer.HoldAuctionsFor(client, instances, repGuids, rules)
				visualization.PrintReport(client, results, repGuids, duration)
			})
		})

		Context("when starting from empty", func() {
			BeforeEach(func() {
				newInstances = map[string]int{
					"green":  100,
					"red":    75,
					"cyan":   50,
					"yellow": 25,
					"gray":   10,
				}
				repDistributions = []int{0, 0, 0, 0, 0, 0, -1, 0, 0, 0, 0, 0, 0, 0, 0, 0, -1, 0, 0, 0}
			})

			It("should distribute evently", func() {
				instances := generateNewInstances(newInstances)
				repGuids, client := generateReps(repDistributions)
				results, duration := auctioneer.HoldAuctionsFor(client, instances, repGuids, rules)
				visualization.PrintReport(client, results, repGuids, duration)
			})
		})
	})
})
