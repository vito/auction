package auction_test

import (
	"time"

	"github.com/onsi/auction/auctioneer"
	"github.com/onsi/auction/instance"
	"github.com/onsi/auction/lossyrep"
	"github.com/onsi/auction/types"
	"github.com/onsi/auction/util"
	"github.com/onsi/auction/visualization"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Ω

var _ = FDescribe("Auction", func() {
	var rules auctioneer.Rules

	var numReps int
	var repResources int
	var initialDistributions map[int][]instance.Instance

	var client types.TestRepPoolClient
	var guids []string

	var numApps int

	generateUniqueInstances := func(numInstances int) []instance.Instance {
		instances := []instance.Instance{}
		for i := 0; i < numInstances; i++ {
			instances = append(instances, instance.New(util.NewGuid("APP"), 1))
		}
		return instances
	}

	randomColor := func() string {
		return []string{"green", "red", "cyan", "yellow", "gray"}[util.R.Intn(5)]
	}

	generateInstancesWithRandomColors := func(numInstances int) []instance.Instance {
		instances := []instance.Instance{}
		for i := 0; i < numInstances; i++ {
			instances = append(instances, instance.New(randomColor(), 1))
		}
		return instances
	}

	generateInstancesForAppGuid := func(numInstances int, appGuid string) []instance.Instance {
		instances := []instance.Instance{}
		for i := 0; i < numInstances; i++ {
			instances = append(instances, instance.New(appGuid, 1))
		}
		return instances
	}

	generateNewColorInstances := func(newInstances map[string]int) []instance.Instance {
		instances := []instance.Instance{}
		for color, num := range newInstances {
			instances = append(instances, generateInstancesForAppGuid(num, color)...)
		}
		return instances
	}

	BeforeEach(func() {
		lossyrep.LatencyMin = 2 * time.Millisecond
		lossyrep.LatencyMax = 12 * time.Millisecond
		lossyrep.Timeout = 50 * time.Millisecond
		lossyrep.Flakiness = 0.95

		util.ResetGuids()
		rules = auctioneer.DefaultRules
		rules.MaxRounds = 100

		numReps = 50
		repResources = 100
		initialDistributions = map[int][]instance.Instance{}
	})

	JustBeforeEach(func() {
		client, guids = buildClient(numReps, repResources)
		for index, instances := range initialDistributions {
			client.SetInstances(guids[index], instances)
		}
	})

	Context("with empty representatives and single-instance apps", func() {
		BeforeEach(func() {
			numApps = 500
		})

		It("should distribute evenly", func() {
			instances := generateUniqueInstances(numApps)

			results, duration := auctioneer.HoldAuctionsFor(client, instances, guids, rules)

			visualization.PrintReport(client, results, guids, duration, rules)
		})
	})

	Context("with non-empty representatives (and single-instance apps)", func() {
		var numApps int
		BeforeEach(func() {
			numApps = 500
			numReps = 20
			initialDistributions[0] = generateUniqueInstances(100)
			initialDistributions[1] = generateUniqueInstances(42)
			initialDistributions[3] = generateUniqueInstances(17)
		})

		It("should distribute evenly", func() {
			instances := generateUniqueInstances(numApps)

			results, duration := auctioneer.HoldAuctionsFor(client, instances, guids, rules)

			visualization.PrintReport(client, results, guids, duration, rules)
		})
	})

	Context("apps with multiple instances", func() {
		var newInstances map[string]int

		Context("when starting from a (terrible) initial distribution", func() {
			BeforeEach(func() {
				numReps = 20

				newInstances = map[string]int{
					"green":  30,
					"red":    27,
					"cyan":   10,
					"yellow": 22,
					"gray":   8,
				}

				initialDistributions[0] = generateInstancesWithRandomColors(100)
				initialDistributions[1] = generateInstancesWithRandomColors(42)
				initialDistributions[3] = generateInstancesWithRandomColors(17)
			})

			It("should distribute evenly", func() {
				instances := generateNewColorInstances(newInstances)
				results, duration := auctioneer.HoldAuctionsFor(client, instances, guids, rules)
				visualization.PrintReport(client, results, guids, duration, rules)
			})
		})

		Context("when starting from empty", func() {
			BeforeEach(func() {
				numReps = 20

				newInstances = map[string]int{
					"green":  100,
					"red":    75,
					"cyan":   50,
					"yellow": 25,
					"gray":   10,
				}
			})

			It("should distribute evently", func() {
				instances := generateNewColorInstances(newInstances)
				results, duration := auctioneer.HoldAuctionsFor(client, instances, guids, rules)
				visualization.PrintReport(client, results, guids, duration, rules)
			})
		})
	})
})
