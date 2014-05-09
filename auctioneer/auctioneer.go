package auctioneer

import (
	"errors"
	"fmt"
	"time"

	"github.com/cheggaaa/pb"
	"github.com/onsi/auction/instance"
	"github.com/onsi/auction/types"
	"github.com/onsi/auction/util"
)

var AllBiddersFull = errors.New("all the bidders were full")

var MaxBiddingPool = 20
var MaxConcurrent = 20

var DefaultRules = Rules{
	MaxRounds:                       100,
	DurationToSleepIfBiddersAreFull: 50 * time.Millisecond,
}

type Rules struct {
	MaxRounds                       int
	DurationToSleepIfBiddersAreFull time.Duration
}

func HoldAuctionsFor(client types.RepPoolClient, instances []instance.Instance, representatives []string, rules Rules) ([]types.AuctionResult, time.Duration) {
	fmt.Printf("\nStarting Auctions\n\n")
	bar := pb.StartNew(len(instances))
	bar.ShowBar = true
	t := time.Now()
	semaphore := make(chan bool, MaxConcurrent)
	c := make(chan types.AuctionResult)
	for _, inst := range instances {
		go func(inst instance.Instance) {
			semaphore <- true
			representatives := randomSubset(representatives, MaxBiddingPool)
			c <- Auction(client, inst, representatives, rules)
			<-semaphore
		}(inst)
	}

	results := []types.AuctionResult{}
	for _ = range instances {
		results = append(results, <-c)
		bar.Increment()
	}

	bar.FinishPrint("\n")

	return results, time.Since(t)
}

func Auction(client types.RepPoolClient, instance instance.Instance, representatives []string, rules Rules) types.AuctionResult {
	var auctionWinner string

	numRounds, numVotes := 0, 0
	t := time.Now()
	for round := 1; round <= rules.MaxRounds; round++ {
		numRounds++
		winner, _, err := vote(client, instance, representatives)
		numVotes += len(representatives)
		if err != nil {
			time.Sleep(rules.DurationToSleepIfBiddersAreFull)
			continue
		}

		c := make(chan types.VoteResult)
		go func() {
			winnerScore, err := client.ReserveAndRecastVote(winner, instance)
			result := types.VoteResult{
				Rep: winner,
			}
			if err != nil {
				result.Error = err.Error()
				c <- result
				return
			}
			result.Score = winnerScore
			c <- result
		}()

		secondRoundVoters := []string{}

		for _, rep := range representatives {
			if rep != winner {
				secondRoundVoters = append(secondRoundVoters, rep)
			}
		}

		_, secondPlaceScore, err := vote(client, instance, secondRoundVoters)

		winnerRecast := <-c
		numVotes += len(representatives)

		if winnerRecast.Error != "" {
			//winner ran out of space on the recast, retry
			continue
		}

		if err == nil && secondPlaceScore < winnerRecast.Score && round < rules.MaxRounds {
			client.Release(winner, instance)
			continue
		}

		client.Claim(winner, instance)
		auctionWinner = winner
		break
	}

	return types.AuctionResult{
		Winner:    auctionWinner,
		Instance:  instance,
		NumRounds: numRounds,
		NumVotes:  numVotes,
		Duration:  time.Since(t),
	}
}

func randomSubset(representatives []string, subsetSize int) []string {
	reps := representatives
	if len(reps) > subsetSize {
		permutation := util.R.Perm(len(representatives))
		reps = []string{}
		for _, index := range permutation[:subsetSize] {
			reps = append(reps, representatives[index])
		}
	}

	return reps
}

func vote(client types.RepPoolClient, instance instance.Instance, representatives []string) (string, float64, error) {
	results := client.Vote(representatives, instance)

	winningScore := 1e9
	winners := []string{}

	for _, result := range results {
		if result.Error != "" {
			continue
		}

		if result.Score < winningScore {
			winningScore = result.Score
			winners = []string{result.Rep}
		} else if result.Score == winningScore { // can be less strict here
			winners = append(winners, result.Rep)
		}
	}

	if len(winners) == 0 {
		return "", 0, AllBiddersFull
	}

	winner := winners[util.R.Intn(len(winners))]

	return winner, winningScore, nil
}
