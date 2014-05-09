package repnatsclient

import (
	"encoding/json"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/cloudfoundry/yagnats"
	"github.com/onsi/auction/instance"
	"github.com/onsi/auction/types"
	"github.com/onsi/auction/util"
)

var TimeoutError = errors.New("timeout")
var RequestFailedError = errors.New("request failed")

type RepClient struct {
	client  yagnats.NATSClient
	timeout time.Duration
}

func New(client yagnats.NATSClient, timeout time.Duration) *RepClient {
	return &RepClient{
		client:  client,
		timeout: timeout,
	}
}

func (rep *RepClient) publishWithTimeout(guid string, subject string, req interface{}, resp interface{}) (err error) {
	replyTo := util.RandomGuid()
	c := make(chan []byte, 1)

	_, err = rep.client.Subscribe(replyTo, func(msg *yagnats.Message) {
		c <- msg.Payload
	})
	if err != nil {
		return err
	}

	payload := []byte{}
	if req != nil {
		payload, err = json.Marshal(req)
		if err != nil {
			return err
		}
	}

	rep.client.PublishWithReplyTo(guid+"."+subject, replyTo, payload)

	select {
	case payload := <-c:
		if string(payload) == "error" {
			return RequestFailedError
		}

		if resp != nil {
			return json.Unmarshal(payload, resp)
		}

		return nil

	case <-time.After(rep.timeout):
		// rep.client.Unsubscribe(sid)
		return TimeoutError
	}
}

func (rep *RepClient) TotalResources(guid string) int {
	var totalResources int
	err := rep.publishWithTimeout(guid, "total_resources", nil, &totalResources)
	if err != nil {
		panic(err)
	}

	return totalResources
}

func (rep *RepClient) Instances(guid string) []instance.Instance {
	var instances []instance.Instance
	err := rep.publishWithTimeout(guid, "instances", nil, &instances)
	if err != nil {
		panic(err)
	}

	return instances
}

func (rep *RepClient) Vote(guids []string, instance instance.Instance) []types.VoteResult {
	replyTo := util.RandomGuid()

	allReceived := new(sync.WaitGroup)
	responses := make(chan types.VoteResult, len(guids))

	_, err := rep.client.Subscribe(replyTo, func(msg *yagnats.Message) {
		defer allReceived.Done()
		var result types.VoteResult
		err := json.Unmarshal(msg.Payload, &result)
		if err != nil {
			return
		}

		responses <- result
	})

	if err != nil {
		return []types.VoteResult{}
	}

	payload, _ := json.Marshal(instance)

	for _, guid := range guids {
		allReceived.Add(1)
		rep.client.PublishWithReplyTo(guid+".vote", replyTo, payload)
	}

	done := make(chan struct{})
	go func() {
		allReceived.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(rep.timeout):
	}

	results := []types.VoteResult{}

	for {
		select {
		case res := <-responses:
			results = append(results, res)
		default:
			return results
		}
	}

	return results
}

func (rep *RepClient) ReserveAndRecastVote(guid string, instance instance.Instance) (float64, error) {
	var score float64
	err := rep.publishWithTimeout(guid, "reserve_and_recast_vote", instance, &score)

	return score, err
}

func (rep *RepClient) Release(guid string, instance instance.Instance) {
	err := rep.publishWithTimeout(guid, "release", instance, nil)
	if err != nil {
		log.Println("failed to release:", err)
	}
}

func (rep *RepClient) Claim(guid string, instance instance.Instance) {
	err := rep.publishWithTimeout(guid, "claim", instance, nil)
	if err != nil {
		log.Println("failed to claim:", err)
	}
}
