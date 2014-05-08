package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"

	"github.com/onsi/auction/instance"
	"github.com/onsi/auction/representative"
)

var listenAddr = flag.String("listenAddr", "", "host:port")
var resources = flag.Int("resources", 100, "total available resources")
var guid = flag.String("guid", "", "guid")

func main() {
	flag.Parse()
	if *guid == "" {
		panic("can haz guid")
	}

	if *listenAddr == "" {
		panic("can haz listen addr")
	}

	rep := representative.New(*guid, *resources, nil)

	http.HandleFunc("/guid", func(w http.ResponseWriter, r *http.Request) {
		// log.Println(*guid, "guid")
		json.NewEncoder(w).Encode(rep.Guid())
	})

	http.HandleFunc("/total_resources", func(w http.ResponseWriter, r *http.Request) {
		// log.Println(*guid, "total resources")
		json.NewEncoder(w).Encode(rep.TotalResources())
	})

	http.HandleFunc("/instances", func(w http.ResponseWriter, r *http.Request) {
		// log.Println(*guid, "instances")
		json.NewEncoder(w).Encode(rep.Instances())
	})

	http.HandleFunc("/vote", func(w http.ResponseWriter, r *http.Request) {
		// log.Println(*guid, "vote")

		var inst instance.Instance

		err := json.NewDecoder(r.Body).Decode(&inst)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		score, err := rep.Vote(inst)
		if err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}

		json.NewEncoder(w).Encode(score)
	})

	http.HandleFunc("/reserve_and_recast_vote", func(w http.ResponseWriter, r *http.Request) {
		// log.Println(*guid, "reserve and recast vote")

		var inst instance.Instance

		err := json.NewDecoder(r.Body).Decode(&inst)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		score, err := rep.ReserveAndRecastVote(inst)
		if err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}

		json.NewEncoder(w).Encode(score)
	})

	http.HandleFunc("/release", func(w http.ResponseWriter, r *http.Request) {
		// log.Println(*guid, "release")

		var inst instance.Instance

		err := json.NewDecoder(r.Body).Decode(&inst)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		rep.Release(inst)

		w.WriteHeader(http.StatusOK)
	})

	http.HandleFunc("/claim", func(w http.ResponseWriter, r *http.Request) {
		// log.Println(*guid, "claim")

		var inst instance.Instance

		err := json.NewDecoder(r.Body).Decode(&inst)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		rep.Claim(inst)

		w.WriteHeader(http.StatusOK)
	})

	fmt.Printf("[%s] serving on %s\n", *guid, *listenAddr)

	panic(http.ListenAndServe(*listenAddr, nil))
}
