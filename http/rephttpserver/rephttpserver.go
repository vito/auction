package rephttpserver

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/onsi/auction/instance"
	"github.com/onsi/auction/representative"
)

func Start(httpAddr string, rep *representative.Representative) {
	http.HandleFunc("/total_resources", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(rep.TotalResources())
	})

	http.HandleFunc("/instances", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(rep.Instances())
	})

	http.HandleFunc("/reset", func(w http.ResponseWriter, r *http.Request) {
		rep.Reset()
	})

	http.HandleFunc("/set_instances", func(w http.ResponseWriter, r *http.Request) {
		var instances []instance.Instance

		err := json.NewDecoder(r.Body).Decode(&instances)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		rep.SetInstances(instances)
	})

	http.HandleFunc("/vote", func(w http.ResponseWriter, r *http.Request) {
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
		var inst instance.Instance

		err := json.NewDecoder(r.Body).Decode(&inst)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		rep.Claim(inst)

		w.WriteHeader(http.StatusOK)
	})

	fmt.Printf("[%s] serving http on %s\n", rep.Guid(), httpAddr)

	panic(http.ListenAndServe(httpAddr, nil))
}
