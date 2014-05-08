package lossyrep

import (
	"errors"
	"time"

	"github.com/onsi/auction/instance"
	"github.com/onsi/auction/representative"
	"github.com/onsi/auction/util"
)

var LatencyMin time.Duration
var LatencyMax time.Duration
var Timeout time.Duration

var Flakiness = 1.0

var TimeoutError = errors.New("timedout")
var FlakyError = errors.New("flakeout")

type LossyRep struct {
	representative.Rep
	Flaky bool
}

func New(totalResources int, flaky bool, instances map[string]instance.Instance) *LossyRep {
	return &LossyRep{
		Rep:   representative.New(totalResources, instances),
		Flaky: flaky,
	}
}

func (rep *LossyRep) Vote(instance instance.Instance) (float64, error) {
	if rep.Flaky {
		if util.Flake(Flakiness) {
			time.Sleep(Timeout)
			return 0, FlakyError
		}
	}
	ok := util.RandomSleep(LatencyMin, LatencyMax, Timeout)
	if !ok {
		return 0, TimeoutError
	}

	return rep.Rep.Vote(instance)
}

func (rep *LossyRep) ReserveAndRecastVote(instance instance.Instance) (float64, error) {
	if rep.Flaky {
		if util.Flake(Flakiness) {
			time.Sleep(Timeout)
			return 0, FlakyError
		}
	}
	ok := util.RandomSleep(LatencyMin, LatencyMax, Timeout)
	if !ok {
		return 0, TimeoutError
	}

	return rep.Rep.ReserveAndRecastVote(instance)
}

func (rep *LossyRep) Release(instance instance.Instance) {
	if rep.Flaky {
		if util.Flake(Flakiness) {
			time.Sleep(Timeout)
			return
		}
	}
	util.RandomSleep(LatencyMin, LatencyMax, Timeout)

	rep.Rep.Release(instance)
}

func (rep *LossyRep) Claim(instance instance.Instance) {
	if rep.Flaky {
		if util.Flake(Flakiness) {
			time.Sleep(Timeout)
			return
		}
	}
	util.RandomSleep(LatencyMin, LatencyMax, Timeout)

	rep.Rep.Claim(instance)
}
