package repnatsclient

import (
	"encoding/json"
	"errors"
	"log"
	"time"

	"github.com/cloudfoundry/yagnats"
	"github.com/onsi/auction/instance"
	"github.com/onsi/auction/util"
)

var TimeoutError = errors.New("timeout")
var RequestFailedError = errors.New("request failed")

type RepClient struct {
	client  yagnats.NATSClient
	guid    string
	timeout time.Duration
}

func New(client yagnats.NATSClient, guid string, timeout time.Duration) *RepClient {
	return &RepClient{
		client:  client,
		guid:    guid,
		timeout: timeout,
	}
}

func (rep *RepClient) Guid() string {
	return rep.guid
}

func (rep *RepClient) publishWithTimeout(subject string, req interface{}, resp interface{}) (err error) {
	replyTo := util.RandomGuid()
	c := make(chan []byte)

	sid, err := rep.client.Subscribe(replyTo, func(msg *yagnats.Message) {
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

	rep.client.PublishWithReplyTo(rep.guid+"."+subject, replyTo, payload)

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
		rep.client.Unsubscribe(sid)
		return TimeoutError
	}
}

func (rep *RepClient) TotalResources() int {
	var totalResources int
	err := rep.publishWithTimeout("total_resources", nil, &totalResources)
	if err != nil {
		panic(err)
	}

	return totalResources
}

func (rep *RepClient) Instances() []instance.Instance {
	var instances []instance.Instance
	err := rep.publishWithTimeout("instances", nil, &instances)
	if err != nil {
		panic(err)
	}

	return instances
}

func (rep *RepClient) Vote(instance instance.Instance) (float64, error) {
	var score float64
	err := rep.publishWithTimeout("vote", instance, &score)

	return score, err
}

func (rep *RepClient) ReserveAndRecastVote(instance instance.Instance) (float64, error) {
	var score float64
	err := rep.publishWithTimeout("reserve_and_recast_vote", instance, &score)

	return score, err
}

func (rep *RepClient) Release(instance instance.Instance) {
	err := rep.publishWithTimeout("release", instance, nil)
	if err != nil {
		log.Println("failed to release:", err)
	}
}

func (rep *RepClient) Claim(instance instance.Instance) {
	err := rep.publishWithTimeout("claim", instance, nil)
	if err != nil {
		log.Println("failed to claim:", err)
	}
}
