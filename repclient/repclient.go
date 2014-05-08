package repclient

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"

	"github.com/onsi/auction/instance"
)

func init() {
	http.DefaultClient.Transport = &http.Transport{
		ResponseHeaderTimeout: 100 * time.Millisecond,
	}
}

type RepClient struct {
	endpoint string
	client   *http.Client
}

func New(endpoint string) *RepClient {
	return &RepClient{
		endpoint: endpoint,
		client:   http.DefaultClient,
	}
}

func (rep *RepClient) Guid() string {
	resp, err := rep.client.Get(rep.endpoint + "/guid")
	if err != nil {
		panic("failed to get guid!")
	}

	defer resp.Body.Close()

	var guid string
	err = json.NewDecoder(resp.Body).Decode(&guid)
	if err != nil {
		panic("invalid guid: " + err.Error())
	}

	return guid
}

func (rep *RepClient) TotalResources() int {
	resp, err := rep.client.Get(rep.endpoint + "/total_resources")
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

func (rep *RepClient) Instances() []instance.Instance {
	resp, err := rep.client.Get(rep.endpoint + "/instances")
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

func (rep *RepClient) Vote(instance instance.Instance) (float64, error) {
	body := new(bytes.Buffer)

	err := json.NewEncoder(body).Encode(instance)
	if err != nil {
		return 0, err
	}

	resp, err := rep.client.Post(rep.endpoint+"/vote", "application/json", body)
	if err != nil {
		// log.Println("voting failed:", err)
		return 0, err
	}

	defer resp.Body.Close()

	var score float64
	err = json.NewDecoder(resp.Body).Decode(&score)
	if err != nil {
		return 0, err
	}

	return score, nil
}

func (rep *RepClient) ReserveAndRecastVote(instance instance.Instance) (float64, error) {
	body := new(bytes.Buffer)

	err := json.NewEncoder(body).Encode(instance)
	if err != nil {
		return 0, err
	}

	resp, err := rep.client.Post(rep.endpoint+"/reserve_and_recast_vote", "application/json", body)
	if err != nil {
		// log.Println("reserving and recasting vote failed:", err)
		return 0, err
	}

	defer resp.Body.Close()

	var score float64
	err = json.NewDecoder(resp.Body).Decode(&score)
	if err != nil {
		return 0, err
	}

	return score, nil
}

func (rep *RepClient) Release(instance instance.Instance) {
	body := new(bytes.Buffer)

	err := json.NewEncoder(body).Encode(instance)
	if err != nil {
		panic("failed to encode instance: " + err.Error())
	}

	resp, err := rep.client.Post(rep.endpoint+"/release", "application/json", body)
	if err != nil {
		// log.Println("releasing failed:", err)
		return
	}

	defer resp.Body.Close()
}

func (rep *RepClient) Claim(instance instance.Instance) {
	body := new(bytes.Buffer)

	err := json.NewEncoder(body).Encode(instance)
	if err != nil {
		panic("failed to encode instance: " + err.Error())
	}

	resp, err := rep.client.Post(rep.endpoint+"/claim", "application/json", body)
	if err != nil {
		// log.Println("claiming failed:", err)
		return
	}

	defer resp.Body.Close()
}
