package main

import (
	"flag"

	"github.com/onsi/auction/http/rephttpserver"
	"github.com/onsi/auction/nats/repnatsserver"
	"github.com/onsi/auction/representative"
)

var resources = flag.Int("resources", 100, "total available resources")
var httpAddr = flag.String("httpAddr", "", "host:port")
var guid = flag.String("guid", "", "guid")
var natsAddr = flag.String("natsAddr", "", "nats server address")

func main() {
	flag.Parse()

	if *guid == "" {
		panic("need guid")
	}

	if *natsAddr == "" && *httpAddr == "" {
		panic("need either nats or http addr (or both)")
	}

	rep := representative.New(*guid, *resources)

	if *natsAddr != "" {
		go repnatsserver.Start(*natsAddr, rep)
	}

	if *httpAddr != "" {
		go rephttpserver.Start(*httpAddr, rep)
	}

	select {}
}
