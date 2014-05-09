package types

import (
	"time"

	"github.com/onsi/auction/instance"
)

type VoteResult struct {
	Rep   string  `json:"r"`
	Score float64 `json:"s"`
	Error string  `json:"e"`
}

type AuctionRequest struct {
	Instance instance.Instance `json:"i"`
	RepGuids []string          `json:"rg"`
	Rules    AuctionRules      `json:"r"`
}

type AuctionResult struct {
	Instance  instance.Instance `json:"i"`
	Winner    string            `json:"w"`
	NumRounds int               `json:"nr"`
	NumVotes  int               `json:"nv"`
	Duration  time.Duration     `json:"d"`
}

type AuctionRules struct {
	MaxRounds        int  `json:"mr"`
	MaxBiddingPool   int  `json:"mb"`
	MaxConcurrent    int  `json:"mc"`
	RepickEveryRound bool `json:"r"`
}

type AuctionCommunicator func(AuctionRequest) AuctionResult

type RepPoolClient interface {
	Vote(guids []string, instance instance.Instance) []VoteResult
	ReserveAndRecastVote(guid string, instance instance.Instance) (float64, error)
	Release(guid string, instance instance.Instance)
	Claim(guid string, instance instance.Instance)
}

type TestRepPoolClient interface {
	RepPoolClient

	TotalResources(guid string) int
	Instances(guid string) []instance.Instance
	SetInstances(guid string, instances []instance.Instance)
	Reset(guid string)
}
