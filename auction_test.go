package auction_test

import (
	"fmt"
	"strings"
	"time"

	"github.com/onsi/auction/auctioneer"
	"github.com/onsi/auction/instance"
	"github.com/onsi/auction/lossyrep"
	"github.com/onsi/auction/representative"
	"github.com/onsi/auction/types"
	"github.com/onsi/auction/util"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Ω

const defaultStyle = "\x1b[0m"
const boldStyle = "\x1b[1m"
const redColor = "\x1b[91m"
const greenColor = "\x1b[32m"
const yellowColor = "\x1b[33m"
const cyanColor = "\x1b[36m"
const grayColor = "\x1b[90m"
const lightGrayColor = "\x1b[37m"

func printReport(client types.RepPoolClient, results []types.AuctionResult, representatives []string, rules auctioneer.Rules, duration time.Duration, color bool) {
	roundsDistribution := map[int]int{}
	auctionedInstances := map[string]bool{}

	///

	fmt.Println("Rounds")
	for _, result := range results {
		roundsDistribution[result.NumRounds] += 1
		auctionedInstances[result.Instance.InstanceGuid] = true
	}

	for i := 1; i <= rules.MaxRounds; i++ {
		if roundsDistribution[i] > 0 {
			fmt.Printf("  %2d: %s\n", i, strings.Repeat("█", roundsDistribution[i]))
		}
	}

	///

	minRounds, maxRounds, totalRounds, meanRounds := 100000000, 0, 0, float64(0)
	for _, result := range results {
		if result.NumRounds < minRounds {
			minRounds = result.NumRounds
		}
		if result.NumRounds > maxRounds {
			maxRounds = result.NumRounds
		}
		totalRounds += result.NumRounds
		meanRounds += float64(result.NumRounds)
	}

	meanRounds = meanRounds / float64(len(results))
	fmt.Printf("  Min: %d | Max: %d | Total: %d | Mean: %.2f\n", minRounds, maxRounds, totalRounds, meanRounds)

	///

	fmt.Println("Votes")
	minVotes, maxVotes, totalVotes, meanVotes := 100000000, 0, 0, float64(0)
	for _, result := range results {
		if result.NumVotes < minVotes {
			minVotes = result.NumVotes
		}
		if result.NumVotes > maxVotes {
			maxVotes = result.NumVotes
		}
		totalVotes += result.NumVotes
		meanVotes += float64(result.NumVotes)
	}

	meanVotes = meanVotes / float64(len(results))
	fmt.Printf("  Min: %d | Max: %d | Total: %d | Mean: %.2f\n", minVotes, maxVotes, totalVotes, meanVotes)

	///

	fmt.Println("Distribution")
	for _, guid := range representatives {
		repString := fmt.Sprintf("%10s", guid)
		lossyRep, ok := client.(*lossyrep.LossyRep)
		if ok && lossyRep.FlakyReps[guid] {
			repString = fmt.Sprintf("%s%10s%s", redColor, repString, defaultStyle)
		}

		instanceString := ""
		instances := client.Instances(guid)
		if color {
			originalCounts := map[string]int{}
			newCounts := map[string]int{}
			for _, instance := range instances {
				if auctionedInstances[instance.InstanceGuid] {
					newCounts[instance.AppGuid] += 1
				} else {
					originalCounts[instance.AppGuid] += 1
				}
			}
			availableColors := []string{"green", "red", "cyan", "yellow", "gray"}
			colorLookup := map[string]string{"red": redColor, "green": greenColor, "cyan": cyanColor, "yellow": yellowColor, "gray": lightGrayColor}
			for _, col := range availableColors {
				instanceString += strings.Repeat(colorLookup[col]+"○"+defaultStyle, originalCounts[col])
				instanceString += strings.Repeat(colorLookup[col]+"●"+defaultStyle, newCounts[col])
			}
			instanceString += strings.Repeat(grayColor+"○"+defaultStyle, client.TotalResources(guid)-len(instances))
		} else {
			numNew := 0
			numOriginal := 0
			for _, instance := range instances {
				if auctionedInstances[instance.InstanceGuid] {
					numNew++
				} else {
					numOriginal++
				}
			}
			instanceString = fmt.Sprintf("%s%s%s", strings.Repeat(lightGrayColor+"●"+defaultStyle, numOriginal), strings.Repeat(greenColor+"●"+defaultStyle, numNew), strings.Repeat(grayColor+"○"+defaultStyle, client.TotalResources(guid)-numOriginal-numNew))
		}

		fmt.Printf("  %s: %s\n", repString, instanceString)
	}

	fmt.Printf("Finished %d Auctions to %d Representatives in %s\n", len(results), len(representatives), duration)
	fmt.Printf("  MaxConcurrent: %d, MaxBiddingBool:%d\n", auctioneer.MaxConcurrent, auctioneer.MaxBiddingPool)
	if _, ok := client.(*lossyrep.LossyRep); ok {
		fmt.Printf("  Latency Range: %s < %s, Timeout: %s, Flakiness: %.2f\n", lossyrep.LatencyMin, lossyrep.LatencyMax, lossyrep.Timeout, lossyrep.Flakiness)
	}

	///

	minTime, maxTime, totalTime, meanTime := time.Hour, time.Duration(0), time.Duration(0), time.Duration(0)
	for _, result := range results {
		if result.Duration < minTime {
			minTime = result.Duration
		}
		if result.Duration > maxTime {
			maxTime = result.Duration
		}
		totalTime += result.Duration
		meanTime += result.Duration
	}

	meanTime = meanTime / time.Duration(len(results))
	fmt.Printf("  Min: %s | Max: %s | Total: %s | Mean: %s\n", minTime, maxTime, totalTime, meanTime)

}

var _ = Describe("Auction", func() {
	var repResources int
	var rules auctioneer.Rules

	BeforeEach(func() {
		lossyrep.LatencyMin = 2 * time.Millisecond
		lossyrep.LatencyMax = 12 * time.Millisecond
		lossyrep.Timeout = 50 * time.Millisecond
		lossyrep.Flakiness = 0.5

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

			printReport(client, results, repGuids, rules, duration, false)
		})
	})

	// Context("with non-empty representatives (and single-instance apps)", func() {
	// 	var numApps int
	// 	var repDistributions []int

	// 	BeforeEach(func() {
	// 		numApps = 100
	// 		repDistributions = []int{100, 20, 10, -7, 19, 32, -42, 71, 10, 20, 13, 82, 36, 42, 16, 13, 28, 57, 12, -2}
	// 	})

	// 	It("should distribute evenly", func() {
	// 		instances := []instance.Instance{}
	// 		for i := 0; i < numApps; i++ {
	// 			instances = append(instances, instance.New(util.NewGuid("APP"), 1))
	// 		}

	// 		var client *lossyrep.LossyRep
	// 		repGuids := []string{}
	// 		repMap := map[string]*representative.Representative{}
	// 		flakyMap := map[string]bool

	// 		for _, repoApps := range repDistributions {
	// 			guid := util.NewGuid("REP")
	// 			numExistingApps := repoApps
	// 			if repoApps < 0 {
	// 				numExistingApps = -repoApps
	// 				flakyMap[guid] = true
	// 			}
	// 			//HERE
	// 			repGuids = append(repGuids, guid)
	// 			repMap[guid] = representative.New(guid, repResources, nil)
	// 		}

	// 		client = lossyrep.New(repMap, flakyMap)

	// 		results, duration := auctioneer.HoldAuctionsFor(client, instances, repGuids, rules)

	// 		printReport(client, results, repGuids, rules, duration, false)

	// 		//NUKE!
	// 		representatives := []representative.Rep{}
	// 		for _, repoApps := range repDistributions {
	// 			numExistingApps := repoApps
	// 			flaky := false
	// 			if repoApps < 0 {
	// 				numExistingApps = -repoApps
	// 				flaky = true
	// 			}
	// 			//FROME HERE
	// 			existingInstances := map[string]instance.Instance{}
	// 			for i := 0; i < numExistingApps; i++ {
	// 				inst := instance.New(util.NewGuid("APP"), 1)
	// 				existingInstances[inst.InstanceGuid] = inst
	// 			}
	// 			representatives = append(representatives, lossyrep.New(repResources, flaky, existingInstances))
	// 		}

	// 		results := auctioneer.HoldAuctionsFor(instances, representatives, rules)

	// 		printReport(results, representatives, rules, false)
	// 	})
	// })

	// Context("when scaling up an app", func() {
	// 	var newInstances map[string]int
	// 	var repDistributions []int

	// 	BeforeEach(func() {
	// 		newInstances = map[string]int{
	// 			"green":  30,
	// 			"red":    27,
	// 			"cyan":   10,
	// 			"yellow": 22,
	// 			"gray":   8,
	// 		}
	// 		repDistributions = []int{100, 20, 10, -7, 19, 32, -42, 71, 10, 20, 13, 82, 36, 42, 16, 13, 28, 57, 12, -2}
	// 	})

	// 	It("should distribute evenly", func() {
	// 		instances := []instance.Instance{}
	// 		for color, num := range newInstances {
	// 			for i := 0; i < num; i++ {
	// 				instances = append(instances, instance.New(color, 1))
	// 			}
	// 		}

	// 		representatives := []representative.Rep{}
	// 		for _, repoApps := range repDistributions {
	// 			numExistingApps := repoApps
	// 			flaky := false
	// 			if repoApps < 0 {
	// 				numExistingApps = -repoApps
	// 				flaky = true
	// 			}
	// 			existingInstances := map[string]instance.Instance{}
	// 			for i := 0; i < numExistingApps; i++ {
	// 				inst := instance.New(util.RandomFrom("green", "red", "yellow", "cyan", "gray"), 1)
	// 				existingInstances[inst.InstanceGuid] = inst
	// 			}
	// 			representatives = append(representatives, lossyrep.New(repResources, flaky, existingInstances))
	// 		}

	// 		results := auctioneer.HoldAuctionsFor(instances, representatives, rules)

	// 		printReport(results, representatives, rules, true)
	// 	})
	// })

	// Context("when starting from empty", func() {
	// 	var newInstances map[string]int
	// 	var repDistributions []int

	// 	BeforeEach(func() {
	// 		newInstances = map[string]int{
	// 			"green":  100,
	// 			"red":    75,
	// 			"cyan":   50,
	// 			"yellow": 25,
	// 			"gray":   10,
	// 		}
	// 		repDistributions = []int{0, 0, 0, 0, 0, 0, -1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	// 	})

	// 	It("should distribute evenly", func() {
	// 		instances := []instance.Instance{}
	// 		for color, num := range newInstances {
	// 			for i := 0; i < num; i++ {
	// 				instances = append(instances, instance.New(color, 1))
	// 			}
	// 		}

	// 		representatives := []representative.Rep{}
	// 		for _, repoApps := range repDistributions {
	// 			numExistingApps := repoApps
	// 			flaky := false
	// 			if repoApps < 0 {
	// 				numExistingApps = -repoApps
	// 				flaky = true
	// 			}
	// 			existingInstances := map[string]instance.Instance{}
	// 			for i := 0; i < numExistingApps; i++ {
	// 				inst := instance.New(util.RandomFrom("green", "red", "yellow", "cyan", "gray"), 1)
	// 				existingInstances[inst.InstanceGuid] = inst
	// 			}
	// 			representatives = append(representatives, lossyrep.New(repResources, flaky, existingInstances))
	// 		}

	// 		results := auctioneer.HoldAuctionsFor(instances, representatives, rules)

	// 		printReport(results, representatives, rules, true)
	// 	})
	// })
})
