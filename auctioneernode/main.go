package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/cloudfoundry/yagnats"
	"github.com/onsi/auction/auctioneer"
	"github.com/onsi/auction/nats/repnatsclient"
	"github.com/onsi/auction/types"
)

var natsAddr = flag.String("natsAddr", "", "nats server address")
var timeout = flag.Duration("timeout", 500*time.Millisecond, "timeout for entire auction")
var maxConcurrent = flag.Int("maxConcurrent", 10, "number of concurrent auctions to hold")

var errorResponse = []byte("error")

func main() {
	flag.Parse()

	if *natsAddr == "" {
		panic("need either nats addr")
	}

	client := yagnats.NewClient()

	err := client.Connect(&yagnats.ConnectionInfo{
		Addr: *natsAddr,
	})

	if err != nil {
		log.Fatalln("no nats:", err)
	}

	semaphore := make(chan bool, *maxConcurrent)

	repclient := repnatsclient.New(client, *timeout)

	client.SubscribeWithQueue("diego.auction", "auction-channel", func(msg *yagnats.Message) {
		semaphore <- true
		defer func() {
			<-semaphore
		}()

		var auctionRequest types.AuctionRequest
		err := json.Unmarshal(msg.Payload, &auctionRequest)
		if err != nil {
			client.Publish(msg.ReplyTo, errorResponse)
			return
		}

		auctionResult := auctioneer.Auction(repclient, auctionRequest)
		payload, _ := json.Marshal(auctionResult)

		client.Publish(msg.ReplyTo, payload)
	})

	fmt.Println("auctioneering")

	select {}
}