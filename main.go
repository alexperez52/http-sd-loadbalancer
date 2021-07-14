package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/go-co-op/gocron"
	"github.com/http-sd-loadbalancer/collector"
	"github.com/http-sd-loadbalancer/config"
	lbdiscovery "github.com/http-sd-loadbalancer/discovery"
	loadbalancer "github.com/http-sd-loadbalancer/loadbalancer"

	"github.com/gorilla/mux"
)

var (
	lb     *loadbalancer.LoadBalancer
	server *http.Server
)

func router() *mux.Router {
	router := mux.NewRouter()
	router.HandleFunc("/jobs", jobHandler).Methods("GET")
	router.HandleFunc("/jobs/{job_id}/targets", targetHandler).Methods("GET")

	return router
}

func jobHandler(w http.ResponseWriter, r *http.Request) {
	displayData := lb.Cache.DisplayJobMapping

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(displayData)
}

func targetHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()["collector_id"]
	params := mux.Vars(r)
	if len(q) == 0 {
		targets := lb.Cache.DisplayCollectorJson[params["job_id"]]
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(targets)

	} else {
		tgs := lb.Cache.DisplayTargetMapping[params["job_id"]+q[0]]
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tgs)
	}
}

func distribute(ctx context.Context) {
	cfg, err := config.Load()
	if err != nil {
		fmt.Println(err)
	}

	// returns the list of collectors based on label selector
	collectors, err := collector.Get(ctx, cfg.LabelSelector)
	if err != nil {
		fmt.Println(err)
	}

	// creates a new discovery manager
	discoveryManager := lbdiscovery.NewManager(ctx)

	// returns the list of targets
	targets, err := lbdiscovery.Get(discoveryManager, cfg)
	if err != nil {
		fmt.Println(err)
	}

	// starts a cronjob to monitor sd targets every 30s
	s := gocron.NewScheduler(time.UTC)
	s.Every(30).Seconds().Do(lbdiscovery.Watch, discoveryManager, &targets)
	s.StartAsync()

	lb = loadbalancer.Init()
	lb.InitializeCollectors(collectors)
	lb.UpdateTargetSet(targets)
	lb.RefreshJobs()

	handler := router()
	server = &http.Server{Addr: ":3030", Handler: handler}
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Error in starting server: %+s\n", err)
		}
	}()
	fmt.Println("Server started...")

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	<-c
	fmt.Println("Server shutting down...")

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != http.ErrServerClosed {
		fmt.Println(err)
	}
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// watcher to monitor file changes in ConfigMap
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		fmt.Println(err)
	}
	defer watcher.Close()

	err = watcher.Add("./conf")
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		for {
			select {
			case event := <-watcher.Events:
				switch event.Op {
				case fsnotify.Write:
					fmt.Println("ConfigMap updated!")
					server.Shutdown(ctx)
					distribute(ctx)
				}
			case err := <-watcher.Errors:
				fmt.Println(err)
			}
		}
	}()

	distribute(ctx)
}
