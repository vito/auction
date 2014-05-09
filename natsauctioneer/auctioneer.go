package natsauctioneer

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/cloudfoundry/yagnats"
	"github.com/onsi/auction/auctioneer"
	"github.com/onsi/auction/instance"
	"github.com/onsi/auction/representative"
	"github.com/onsi/auction/util"
)

var AllBiddersFull = errors.New("all the bidders were full")

var MaxBiddingPool = 20
var MaxConcurrent = 20

type voteResponse struct {
	representative representative.Rep
	score          float64
	err            error
}

func HoldAuctionsFor(natsClient yagnats.NATSClient, instances []instance.Instance, representatives []representative.Rep, rules auctioneer.Rules) []auctioneer.AuctionResult {
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
			c <- Auction(natsClient, 50*time.Millisecond, inst, reps, rules)
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

func Auction(natsClient yagnats.NATSClient, timeout time.Duration, instance instance.Instance, representatives []representative.Rep, rules auctioneer.Rules) auctioneer.AuctionResult {
	var auctionWinner representative.Rep
	numRounds, numVotes := 0, 0
	t := time.Now()
	for round := 1; round <= rules.MaxRounds; round++ {
		numRounds++
		winner, _, err := vote(natsClient, timeout, instance, representatives, nil)
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

		_, secondPlaceScore, err := vote(natsClient, timeout, instance, representatives, winner)

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

	return auctioneer.AuctionResult{
		Winner:    auctionWinner,
		Instance:  instance,
		NumRounds: numRounds,
		NumVotes:  numVotes,
		Duration:  time.Since(t),
	}
}

type VoteMessage struct {
	Exclude  string
	Instance instance.Instance
}

type VoteResponse struct {
	Guid  string
	Score float64
	Error string
}

func vote(natsClient yagnats.NATSClient, timeout time.Duration, instance instance.Instance, representatives []representative.Rep, skip representative.Rep) (representative.Rep, float64, error) {
	replyTo := util.RandomGuid()

	allReceived := new(sync.WaitGroup)
	responses := make(chan voteResponse, len(representatives))

	_, err := natsClient.Subscribe(replyTo, func(msg *yagnats.Message) {
		defer allReceived.Done()

		var resp VoteResponse
		err := json.Unmarshal(msg.Payload, &resp)
		if err != nil {
			log.Println("BOGUS, MAN", string(msg.Payload))
			return
		}

		res := voteResponse{
			score: resp.Score,
		}

		for _, rep := range representatives {
			if rep.Guid() == resp.Guid {
				res.representative = rep
			}
		}

		if resp.Error != "" {
			res.err = errors.New(resp.Error)
		}

		responses <- res
	})
	if err != nil {
		return nil, 0, err
	}

	msg := VoteMessage{Instance: instance}
	if skip != nil {
		msg.Exclude = skip.Guid()
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		return nil, 0, err
	}

	for _, rep := range representatives {
		if skip != nil && rep.Guid() == skip.Guid() {
			continue
		}
		allReceived.Add(1)
		natsClient.PublishWithReplyTo(rep.Guid()+".auction", replyTo, payload)
	}

	done := make(chan struct{})
	go func() {
		allReceived.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(timeout):
	}

	allResponses := readResponses(responses)

	// util.RandomSleep(1*time.Millisecond, 5*time.Millisecond, 50*time.Millisecond)

	winningScore := 1e9
	winners := []representative.Rep{}

	for _, vote := range allResponses {
		if vote.err != nil {
			continue
		}

		if vote.score < winningScore {
			winningScore = vote.score
			winners = []representative.Rep{vote.representative}
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

func readResponses(responses <-chan voteResponse) []voteResponse {
	read := []voteResponse{}

	for {
		select {
		case res := <-responses:
			read = append(read, res)
		default:
			return read
		}
	}
}
