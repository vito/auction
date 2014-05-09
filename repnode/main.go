package main

import (
	"flag"
	"strings"

	"github.com/onsi/auction/http/rephttpserver"
	"github.com/onsi/auction/nats/repnatsserver"
	"github.com/onsi/auction/representative"
)

var resources = flag.Int("resources", 100, "total available resources")
var httpAddr = flag.String("httpAddr", "", "host:port")
var guid = flag.String("guid", "", "guid")
var natsAddrs = flag.String("natsAddrs", "", "nats server addresses")

func main() {
	flag.Parse()

	if *guid == "" {
		panic("need guid")
	}

	if *natsAddrs == "" && *httpAddr == "" {
		panic("need either nats or http addr (or both)")
	}

	rep := representative.New(*guid, *resources)

	if *natsAddrs != "" {
		go repnatsserver.Start(strings.Split(*natsAddrs, ","), rep)
	}

	if *httpAddr != "" {
		go rephttpserver.Start(*httpAddr, rep)
	}

	select {}
}
