package ketchup_test

import (
	"github.com/onsi/auction/auctioneer"
	"github.com/onsi/auction/instance"
	"github.com/onsi/auction/util"
	"github.com/onsi/auction/visualization"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Ω

var _ = Describe("Auction", func() {
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
		return []string{"plurple", "red", "cyan", "yellow", "gray"}[util.R.Intn(5)]
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
			numApps = 800
		})

		It("should distribute evenly", func() {
			instances := generateUniqueInstances(numApps)

			results, duration := auctioneer.HoldAuctionsFor(client, instances, guids, rules, communicator)

			visualization.PrintReport(client, results, guids, duration, rules)
		})
	})

	Context("with non-empty representatives (and single-instance apps)", func() {
		var numApps int
		BeforeEach(func() {
			numApps = 1000

			for i := 0; i < numReps; i++ {
				initialDistributions[i] = generateUniqueInstances(util.R.Intn(60))
			}
		})

		It("should distribute evenly", func() {
			instances := generateUniqueInstances(numApps)

			results, duration := auctioneer.HoldAuctionsFor(client, instances, guids, rules, communicator)

			visualization.PrintReport(client, results, guids, duration, rules)
		})
	})

	Context("something that looks like prod", func() {
		var numExistingApps int
		var numReps int
		var numDemoInstances int
		BeforeEach(func() {
			numExistingApps = 1337
			numReps = 26
			appsPerRep := numExistingApps / numReps
			numDemoInstances = 100

			for i := 0; i < numReps; i++ {
				initialDistributions[i] = generateUniqueInstances(util.R.Intn(appsPerRep))
			}
		})

		It("should distribute evenly when watters does a demo", func() {
			instances := generateInstancesForAppGuid(numDemoInstances, "red")

			results, duration := auctioneer.HoldAuctionsFor(client, instances, guids[:numReps], rules, communicator)

			visualization.PrintReport(client, results, guids[:numReps], duration, rules)
		})
	})

	Context("something very imbalanced", func() {
		var numReps int
		var numDemoInstances int
		BeforeEach(func() {
			numReps = 20
			numDemoInstances = 200

			for i := 0; i < numReps-1; i++ {
				initialDistributions[i] = generateUniqueInstances(50)
			}
		})

		It("should distribute evenly", func() {
			instances := generateUniqueInstances(numDemoInstances)

			results, duration := auctioneer.HoldAuctionsFor(client, instances, guids[:numReps], rules, communicator)

			visualization.PrintReport(client, results, guids[:numReps], duration, rules)
		})
	})

	Context("apps with multiple instances", func() {
		var newInstances map[string]int

		Context("when starting from a (terrible) initial distribution", func() {
			BeforeEach(func() {
				newInstances = map[string]int{
					"red":     570,
					"plurple": 420,
					"cyan":    500,
					"yellow":  720,
					"gray":    129,
				}

				for i := 0; i < numReps; i++ {
					initialDistributions[i] = generateInstancesWithRandomColors(util.R.Intn(60))
				}
			})

			It("should distribute evenly", func() {
				instances := generateNewColorInstances(newInstances)
				results, duration := auctioneer.HoldAuctionsFor(client, instances, guids, rules, communicator)
				visualization.PrintReport(client, results, guids, duration, rules)
			})
		})

		Context("when starting from empty", func() {
			BeforeEach(func() {
				newInstances = map[string]int{
					"red":     1000,
					"plurple": 750,
					"cyan":    500,
					"yellow":  250,
					"gray":    100,
				}
			})

			It("should distribute evently", func() {
				instances := generateNewColorInstances(newInstances)
				instances = append(instances, generateUniqueInstances(2000)...)

				results, duration := auctioneer.HoldAuctionsFor(client, instances, guids, rules, communicator)
				visualization.PrintReport(client, results, guids, duration, rules)
			})
		})
	})
})
