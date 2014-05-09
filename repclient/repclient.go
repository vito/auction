package repclient

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/onsi/auction/instance"
	"github.com/onsi/auction/types"
)

var semaphore chan bool
var MaxConcurrentConnections = 100

func init() {
	semaphore = make(chan bool, MaxConcurrentConnections)
	http.DefaultClient.Transport = &http.Transport{
		ResponseHeaderTimeout: 100 * time.Millisecond,
	}
}

type RepClient struct {
	endpoints map[string]string
	client    *http.Client
}

func New(endpoints map[string]string) *RepClient {
	return &RepClient{
		endpoints: endpoints,
		client:    http.DefaultClient,
	}
}

func (rep *RepClient) enter() {
	semaphore <- true
}

func (rep *RepClient) exit() {
	<-semaphore
}

func (rep *RepClient) TotalResources(guid string) int {
	rep.enter()
	defer rep.exit()

	resp, err := rep.client.Get(rep.endpoints[guid] + "/total_resources")
	if err != nil {
		panic("failed to get total resources!")
	}

	defer resp.Body.Close()

	var totalResources int
	err = json.NewDecoder(resp.Body).Decode(&totalResources)
	if err != nil {
		panic("invalid total resources: " + err.Error())
	}

	return totalResources
}

func (rep *RepClient) Instances(guid string) []instance.Instance {
	rep.enter()
	defer rep.exit()

	resp, err := rep.client.Get(rep.endpoints[guid] + "/instances")
	if err != nil {
		panic("failed to get instances!")
	}

	defer resp.Body.Close()

	var instances []instance.Instance
	err = json.NewDecoder(resp.Body).Decode(&instances)
	if err != nil {
		panic("invalid instances: " + err.Error())
	}

	return instances
}

func (rep *RepClient) vote(guid string, instance instance.Instance, c chan types.VoteResult) {
	rep.enter()
	defer rep.exit()
	result := types.VoteResult{
		Rep: guid,
	}
	defer func() {
		c <- result
	}()

	body := new(bytes.Buffer)
	err := json.NewEncoder(body).Encode(instance)
	if err != nil {
		result.Error = err.Error()
		return
	}

	resp, err := rep.client.Post(rep.endpoints[guid]+"/vote", "application/json", body)
	if err != nil {
		println(err.Error())
		result.Error = err.Error()
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		result.Error = "failed"
		return
	}

	var score float64
	err = json.NewDecoder(resp.Body).Decode(&score)
	if err != nil {
		result.Error = err.Error()
		return
	}
	result.Score = score

	return
}

func (rep *RepClient) Vote(guids []string, instance instance.Instance) []types.VoteResult {
	c := make(chan types.VoteResult)
	for _, guid := range guids {
		go rep.vote(guid, instance, c)
	}

	results := []types.VoteResult{}
	for _ = range guids {
		results = append(results, <-c)
	}

	return results
}

func (rep *RepClient) ReserveAndRecastVote(guid string, instance instance.Instance) (float64, error) {
	rep.enter()
	defer rep.exit()

	body := new(bytes.Buffer)

	err := json.NewEncoder(body).Encode(instance)
	if err != nil {
		return 0, err
	}

	resp, err := rep.client.Post(rep.endpoints[guid]+"/reserve_and_recast_vote", "application/json", body)
	if err != nil {
		return 0, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, errors.New("failed")
	}

	var score float64
	err = json.NewDecoder(resp.Body).Decode(&score)
	if err != nil {
		return 0, err
	}

	return score, nil
}

func (rep *RepClient) Release(guid string, instance instance.Instance) {
	rep.enter()
	defer rep.exit()

	body := new(bytes.Buffer)

	err := json.NewEncoder(body).Encode(instance)
	if err != nil {
		panic("failed to encode instance: " + err.Error())
	}

	resp, err := rep.client.Post(rep.endpoints[guid]+"/release", "application/json", body)
	if err != nil {
		return
	}

	resp.Body.Close()
}

func (rep *RepClient) Claim(guid string, instance instance.Instance) {
	rep.enter()
	defer rep.exit()

	body := new(bytes.Buffer)

	err := json.NewEncoder(body).Encode(instance)
	if err != nil {
		panic("failed to encode instance: " + err.Error())
	}

	resp, err := rep.client.Post(rep.endpoints[guid]+"/claim", "application/json", body)
	if err != nil {
		return
	}

	resp.Body.Close()
}
