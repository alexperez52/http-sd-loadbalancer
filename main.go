package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-co-op/gocron"
	"github.com/http-sd-loadbalancer/collector"
	"github.com/http-sd-loadbalancer/config"
	lbdiscovery "github.com/http-sd-loadbalancer/discovery"
	loadbalancer "github.com/http-sd-loadbalancer/loadbalancer"

	"github.com/gorilla/mux"
)

var lb *loadbalancer.LoadBalancer

func main() {

	// Create new disocvery manager
	discoveryManager := lbdiscovery.NewManager()

	// Load the ConfigMap
	var cfg = config.Load()

	collectors, err := collector.Get(context.Background(), cfg.LabelSelector)
	if err != nil {
		fmt.Println(err)
	}

	// Obtain initial TargetGroups using service discovery
	targetMapping, err := lbdiscovery.Get(discoveryManager, cfg)
	if err != nil {
		fmt.Println(err)
	}

	// Format initial TargetGroups into list of targets
	targetList := lbdiscovery.GetTargetList(targetMapping)
	lb = loadbalancer.Init()

	// Start cronjob to to run service dicsovery at fixed intervals (30s)
	s := gocron.NewScheduler(time.UTC)
	s.Every(30).Seconds().Do(lbdiscovery.Watch, discoveryManager, &targetList)

	s.StartAsync()

	lb.InitializeCollectors(collectors)
	lb.UpdateTargetSet(targetList)
	lb.RefreshJobs()

	// Starting server
	fmt.Println("Server started...")
	router := mux.NewRouter()
	router.HandleFunc("/jobs", DisplayAll).Methods("GET")
	router.HandleFunc("/jobs/{job_id}/targets", DisplayCollectorMapping).Methods("GET")
	http.ListenAndServe(":3030", router)
}

// Exposes all current jobs with their corresponding targets link
func DisplayAll(w http.ResponseWriter, r *http.Request) {

	displayData := loadbalancer.SetupDisplayData(lb)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(displayData)
}

// Exposes the scrape targets on the appropriate end points
// If there is > 0 query params under the key 'collector_id' then it will only expose targets for that collector.
// Otherwise it will jsut expose all collector's jobs
func DisplayCollectorMapping(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()["collector_id"]
	if len(q) == 0 {
		params := mux.Vars(r)
		targets := loadbalancer.SetupDisplayTargets(lb, params)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(targets)

	} else {
		tgs := loadbalancer.SetupCollectorData(lb, q)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tgs)
	}
}
