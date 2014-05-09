package auction_test

import (
	"github.com/onsi/auction/auctioneer"
	"github.com/onsi/auction/instance"
	"github.com/onsi/auction/util"
	"github.com/onsi/auction/visualization"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Ω

var _ = FDescribe("Auction", func() {
	var initialDistributions map[int][]instance.Instance

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
		util.ResetGuids()
		initialDistributions = map[int][]instance.Instance{}
	})

	JustBeforeEach(func() {
		for index, instances := range initialDistributions {
			client.SetInstances(guids[index], instances)
		}
	})

	Context("with empty representatives and single-instance apps", func() {
		BeforeEach(func() {
			numApps = 300
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
			numApps = 100
			initialDistributions[0] = generateUniqueInstances(0)
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
