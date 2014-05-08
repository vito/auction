package auction_test

import (
	"fmt"
	"strings"
	"time"

	"github.com/onsi/auction/auctioneer"
	"github.com/onsi/auction/instance"
	"github.com/onsi/auction/representative"
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

func printReport(results []auctioneer.AuctionResult, representatives []*representative.Representative, rules auctioneer.Rules) {
	roundsDistribution := map[int]int{}
	auctionedInstances := map[string]bool{}
	fmt.Println("Stats")
	fmt.Printf("  %d Auctions to %d Representatives\n", len(results), len(representatives))
	fmt.Printf("  Latency Range: %s < %s\n", representative.LatencyMin, representative.LatencyMax)
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

	fmt.Println("Auctions")
	for _, result := range results {
		roundsDistribution[result.NumRounds] += 1
		auctionedInstances[result.Instance.InstanceGuid] = true
	}

	for i := 1; i <= rules.MaxRounds; i++ {
		if roundsDistribution[i] > 0 {
			fmt.Printf("  %2d: %s\n", i, strings.Repeat("█", roundsDistribution[i]))
		}
	}

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

	fmt.Println("Distribution")
	for _, rep := range representatives {
		instances := rep.Instances()
		numNew := 0
		numOriginal := 0
		for _, instance := range instances {
			if auctionedInstances[instance.InstanceGuid] {
				numNew++
			} else {
				numOriginal++
			}
		}

		fmt.Printf("  %6s: %s%s%s\n", rep.Guid, strings.Repeat(lightGrayColor+"●"+defaultStyle, numOriginal), strings.Repeat(greenColor+"●"+defaultStyle, numNew), strings.Repeat(grayColor+"○"+defaultStyle, rep.TotalResources-numOriginal-numNew))
	}
}

var _ = Describe("Auction", func() {
	var repResources int
	var rules auctioneer.Rules

	BeforeEach(func() {
		representative.LatencyMin = 10 * time.Millisecond
		representative.LatencyMax = 50 * time.Millisecond
		representative.Timeout = 30 * time.Millisecond

		repResources = 100
		util.ResetGuids()
		rules = auctioneer.DefaultRules
		rules.MaxRounds = 100
	})

	Context("with empty representatives and single-instance apps", func() {
		var numApps int
		var numReps int

		BeforeEach(func() {
			numApps = 400
			numReps = 5
		})

		It("should distribute evenly", func() {
			instances := []instance.Instance{}
			for i := 0; i < numApps; i++ {
				instances = append(instances, instance.New(util.NewGuid("APP"), 1))
			}

			representatives := []*representative.Representative{}
			for i := 0; i < numReps; i++ {
				representatives = append(representatives, representative.New(repResources, nil))
			}

			results := auctioneer.HoldAuctionsFor(instances, representatives, rules)

			printReport(results, representatives, rules)
		})
	})

	Context("with non-empty representatives (and single-instance apps)", func() {
		var numApps int
		var repDistributions []int

		BeforeEach(func() {
			numApps = 100
			repDistributions = []int{0, 50}
		})

		It("should distribute evenly", func() {
			instances := []instance.Instance{}
			for i := 0; i < numApps; i++ {
				instances = append(instances, instance.New(util.NewGuid("APP"), 1))
			}

			representatives := []*representative.Representative{}
			for _, numExistingApps := range repDistributions {
				existingInstances := map[string]instance.Instance{}
				for i := 0; i < numExistingApps; i++ {
					inst := instance.New(util.NewGuid("APP"), 1)
					existingInstances[inst.InstanceGuid] = inst
				}
				representatives = append(representatives, representative.New(repResources, existingInstances))
			}

			results := auctioneer.HoldAuctionsFor(instances, representatives, rules)

			printReport(results, representatives, rules)
		})
	})
})
