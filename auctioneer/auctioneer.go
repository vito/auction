package auctioneer

import (
	"errors"
	"time"

	"github.com/onsi/auction/instance"
	"github.com/onsi/auction/representative"
	"github.com/onsi/auction/util"
)

var AllBiddersFull = errors.New("all the bidders were full")
var DefaultRules = Rules{
	MaxRounds:                       10,
	DurationToSleepIfBiddersAreFull: 50 * time.Millisecond,
}

type Rules struct {
	MaxRounds                       int
	DurationToSleepIfBiddersAreFull time.Duration
}

type voteResponse struct {
	representative *representative.Representative
	score          float64
	err            error
}

type AuctionResult struct {
	Instance  instance.Instance
	Winner    *representative.Representative
	NumRounds int
	NumVotes  int
	Duration  time.Duration
}

func HoldAuctionsFor(instances []instance.Instance, representatives []*representative.Representative, rules Rules) []AuctionResult {
	c := make(chan AuctionResult)
	for _, inst := range instances {
		go func(inst instance.Instance) {
			util.RandomSleep(10*time.Millisecond, 50*time.Millisecond, 50*time.Millisecond)
			c <- Auction(inst, representatives, rules)
		}(inst)
	}

	results := []AuctionResult{}
	for _ = range instances {
		results = append(results, <-c)
	}

	return results
}

func Auction(instance instance.Instance, representatives []*representative.Representative, rules Rules) AuctionResult {
	var auctionWinner *representative.Representative
	numRounds, numVotes := 0, 0
	t := time.Now()
	for round := 1; round <= rules.MaxRounds; round++ {
		numRounds++
		winner, _, err := vote(instance, representatives, nil)
		numVotes += len(representatives)
		if err != nil {
			time.Sleep(rules.DurationToSleepIfBiddersAreFull)
			continue
		}

		c := make(chan voteResponse)
		go func() {
			winnerScore, err := winner.ReserveAndRecastVote(instance)
			c <- voteResponse{
				representative: winner,
				score:          winnerScore,
				err:            err,
			}
		}()

		_, secondPlaceScore, err := vote(instance, representatives, winner)

		winnerRecast := <-c
		numVotes += len(representatives)

		if winnerRecast.err != nil {
			//winner ran out of space on the recast, retry
			continue
		}

		if err == nil && secondPlaceScore < winnerRecast.score && round < rules.MaxRounds {
			winner.Release(instance)
			continue
		}

		winner.Claim(instance)
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

func vote(instance instance.Instance, representatives []*representative.Representative, skip *representative.Representative) (*representative.Representative, float64, error) {
	c := make(chan voteResponse)
	n := 0

	for _, rep := range representatives {
		if rep == skip {
			continue
		}
		n++
		go func(rep *representative.Representative) {
			score, err := rep.Vote(instance)
			c <- voteResponse{
				representative: rep,
				score:          score,
				err:            err,
			}
		}(rep)
	}

	winningScore := 1e9
	winners := []*representative.Representative{}

	for i := 0; i < n; i++ {
		vote := <-c
		if vote.err != nil {
			continue
		}

		if vote.score < winningScore {
			winningScore = vote.score
			winners = []*representative.Representative{vote.representative}
		} else if vote.score == winningScore { // can be less strict here
			winners = append(winners, vote.representative)
		}
	}

	if len(winners) == 0 {
		return nil, 0, AllBiddersFull
	}

	winner := winners[util.R.Intn(len(winners))]

	return winner, winningScore, nil
}
