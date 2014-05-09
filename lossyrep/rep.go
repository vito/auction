package lossyrep

import (
	"errors"
	"time"

	"github.com/onsi/auction/auctioneer"
	"github.com/onsi/auction/instance"
	"github.com/onsi/auction/representative"
	"github.com/onsi/auction/util"
)

var LatencyMin time.Duration
var LatencyMax time.Duration
var Timeout time.Duration

var Flakiness = 1.0

var FlakyError = errors.New("flakeout")

type LossyRep struct {
	reps      map[string]*representative.Representative
	FlakyReps map[string]bool
}

func New(reps map[string]*representative.Representative, flakyReps map[string]bool) *LossyRep {
	return &LossyRep{
		reps:      reps,
		FlakyReps: flakyReps,
	}
}

func (rep *LossyRep) beSlowAndFlakey(guid string) bool {
	if rep.FlakyReps[guid] {
		if util.Flake(Flakiness) {
			time.Sleep(Timeout)
			return true
		}
	}
	ok := util.RandomSleep(LatencyMin, LatencyMax, Timeout)
	if !ok {
		return true
	}

	return false
}

func (rep *LossyRep) TotalResources(guid string) int {
	return rep.reps[guid].TotalResources()
}

func (rep *LossyRep) Instances(guid string) []instance.Instance {
	return rep.reps[guid].Instances()
}

func (rep *LossyRep) Vote(representatives []string, instance instance.Instance) []auctioneer.VoteResult {
	c := make(chan auctioneer.VoteResult)
	for _, guid := range representatives {
		go func(guid string) {
			if rep.beSlowAndFlakey(guid) {
				c <- auctioneer.VoteResult{}
			}
			score, err := rep.Reps[guid].Vote(instance)
			if err != nil {
				c <- auctioneer.VoteResult{}
			}
			c <- auctioneer.VoteResult{
				Rep:   guid,
				Score: score,
			}
		}(guid)
	}

	results := []auctioneer.VoteResult{}
	for _ := range representatives {
		voteResult := <-c
		if voteResult.Rep != "" {
			results = append(results, voteResult)
		}
	}

	return results
}

func (rep *LossyRep) ReserveAndRecastVote(guid string, instance instance.Instance) (float64, error) {
	if rep.beSlowAndFlakey(guid) {
		return 0, FlakyError
	}

	return rep.reps[guid].ReserveAndRecastVote(instance)
}

func (rep *LossyRep) Release(guid string, instance instance.Instance) {
	rep.beSlowAndFlakey(guid)

	rep.reps[guid].Release(instance)
}

func (rep *LossyRep) Claim(guid string, instance instance.Instance) {
	rep.beSlowAndFlakey(guid)

	rep.reps[guid].Claim(instance)
}
