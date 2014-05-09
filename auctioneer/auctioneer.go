package auctioneer

import (
	"errors"
	"fmt"
	"time"

	"github.com/onsi/auction/auctioneer"
	"github.com/onsi/auction/instance"
	"github.com/onsi/auction/representative"
	"github.com/onsi/auction/util"
)

type RepPoolClient interface {
	TotalResources(guid string) int
	Instances(guid string) []instance.Instance

	Vote(guids []string, instance instance.Instance) []VoteResult
	ReserveAndRecastVote(guid string, instance instance.Instance) (float64, error)
	Release(guid string, instance instance.Instance)
	Claim(guid string, instance instance.Instance)
}

var AllBiddersFull = errors.New("all the bidders were full")

var MaxBiddingPool = 20
var MaxConcurrent = 20

var DefaultRules = Rules{
	MaxRounds:                       10,
	DurationToSleepIfBiddersAreFull: 50 * time.Millisecond,
}

type Rules struct {
	MaxRounds                       int
	DurationToSleepIfBiddersAreFull time.Duration
}

type VoteResult struct {
	Rep   string
	Score float64
}

type AuctionResult struct {
	Instance  instance.Instance
	Winner    string
	NumRounds int
	NumVotes  int
	Duration  time.Duration
}

func HoldAuctionsFor(client RepPoolClient, instances []instance.Instance, representatives []string, rules Rules) []AuctionResult {
	t := time.Now()
	semaphore := make(chan bool, MaxConcurrent)
	c := make(chan auctioneer.AuctionResult)
	for _, inst := range instances {
		go func(inst instance.Instance) {
			semaphore <- true
			reps := representatives
			if len(representatives) > MaxBiddingPool {
				permutation := util.R.Perm(len(representatives))
				reps = []representative.Rep{}
				for _, index := range permutation[:MaxBiddingPool] {
					reps = append(reps, representatives[index])
				}
			}
			c <- Auction(client, inst, reps, rules)
			<-semaphore
		}(inst)
	}

	results := []auctioneer.AuctionResult{}
	for _ = range instances {
		results = append(results, <-c)
	}

	fmt.Printf("Auction took: %s\n", time.Since(t))

	return results
}

func Auction(client RepPoolClient, instance instance.Instance, representatives []string, rules Rules) AuctionResult {
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

		c := make(chan voteResponse)
		go func() {
			winnerScore, err := client.ReserveAndRecastVote(winner, instance)
			c <- voteResponse{
				representative: winner,
				score:          winnerScore,
				err:            err,
			}
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

		if winnerRecast.err != nil {
			//winner ran out of space on the recast, retry
			continue
		}

		if err == nil && secondPlaceScore < winnerRecast.score && round < rules.MaxRounds {
			client.Release(winner, instance)
			continue
		}

		client.Claim(winner, instance)
		auctionWinner = winner
		break
	}

	return AuctionResult{
		Winner:    auctionWinner,
		Instance:  instance,
		NumRounds: numRounds,
		NumVotes:  numVotes,
		Duration:  time.Since(t),
	}
}

func vote(client RepPoolClient, instance instance.Instance, representatives []string) (string, float64, error) {
	c := make(chan voteResponse)
	n := 0

	results := client.Vote(representatives, instance)

	winningScore := 1e9
	winners := []string{}

	for _, result := range results {
		if result.Score < winningScore {
			winningScore = result.Score
			winners = []string{result.Rep}
		} else if result.Score == winningScore { // can be less strict here
			winners = append(winners, result.Rep)
		}
	}

	if len(winners) == 0 {
		return nil, 0, AllBiddersFull
	}

	winner := winners[util.R.Intn(len(winners))]

	return winner, winningScore, nil

	// for _, rep := range representatives {
	// 	if rep == skip {
	// 		continue
	// 	}
	// 	n++
	// 	go func(rep representative.Rep) {
	// 		score, err := rep.Vote(instance)
	// 		c <- voteResponse{
	// 			representative: rep,
	// 			score:          score,
	// 			err:            err,
	// 		}
	// 	}(rep)
	// }
}
