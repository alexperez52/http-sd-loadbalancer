package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/http-sd-loadbalancer/config"
	lbdiscovery "github.com/http-sd-loadbalancer/discovery"

	"github.com/gorilla/mux"
	"github.com/prometheus/common/model"
)

/*
	Load balancer will serve on an HTTP server exposing /jobs/<job_id>/targets <- these are configured using least connection
	Load balancer will need information about the collectors in order to set the URLs
	Keep a Map of what each collector currently holds and update it based on new scrape target updates
*/
// Create a struct that holds collector - and jobs for that collector
// This struct will be parsed into endpoint with collector and jobs info

type Collector struct {
	name     string
	numTargs int
}

// Label to display on the http server
type LinkLabel struct {
	Link string `json:"_link"`
}

type TargetData struct {
	JobName string
	Target  string
	Labels  model.LabelSet
}

type CollectorJson struct {
	Link string                    `json:"_link"`
	Jobs []lbdiscovery.TargetGroup `json:"targets"`
}

var (
	// ErrInvalidLBYAML represents an error in the format of the original YAML configuration file.
	ErrInvalidLBYAML = errors.New("couldn't parse the loadbalancer configuration")
	// ErrInvalidLBFile represents an error in reading the original YAML configuration file.
	ErrInvalidLBFile = errors.New("couldn't read the loadbalancer configuration file")
)

// Next will hold the next collector pointer to be used when adding a new job (Uses least connection to be determined)
type Next struct {
	nextCollector *Collector
}

var next = Next{}

// -------------------------- Mock Data ------------------------------------
// Mock list of targets

var collectors = []string{"collector-1", "collector-2", "collector-3"}

// -------------------------- End Mock Data --------------------------------

var targetSet = make(map[string]lbdiscovery.TargetData) //set of targets - periodically updated // Once configured it will be updated with service discovery

var targetMap = make(map[string]lbdiscovery.TargetData) //key=target, value=collectorName

var colMap = make(map[string]*Collector) // key=collectorName, value=Collector{}

var displayData = make(map[string]LinkLabel) // This is for the DisplayAll func

var displayData2 = make(map[string]CollectorJson) // This is for the DisplayCollectorMapping func

var targetList []lbdiscovery.TargetData
var targetItemMap = make(map[string]*TargetItem) // key=collectorName, value=Collector{}

type TargetItem struct {
	JobName      string
	Link         LinkLabel
	TargetUrl    string
	Label        model.LabelSet
	CollectorPtr *Collector
}

func main() {

	// TODO: Use service discovery instead of mock data && reformat structs for better performance / cleaner code
	// Load the ConfigMap
	var cfg = config.Load()

	// Create new disocvery manager
	discoveryManager := lbdiscovery.NewManager()

	// Obtain TargetGroups using service discovery
	targetMapping, err := lbdiscovery.Get(discoveryManager, cfg)
	if err != nil {
		fmt.Println(err)
	}

	// Format TargetGroups into list of targets
	targetList = lbdiscovery.GetTargetList(targetMapping)

	InitializeCollectors()
	UpdateTargetSet()

	// The following 2 function calls will reconcile the scrape targets.
	// RemoveOutdatedJobs will compare internal map to the dynamically changing targetMap to remove any targets that are no longer being used
	// AddUpdatedJobs will compare internal map to the dynamically changing targetMap to add any new targets
	RemoveOutdatedJobs()
	AddUpdatedJobs()
	router := mux.NewRouter()
	router.HandleFunc("/jobs", DisplayAll).Methods("GET")
	router.HandleFunc("/jobs/{job_id}/targets", DisplayCollectorMapping).Methods("GET")
	http.ListenAndServe(":3030", router)
	fmt.Println("Server started...")
}

func DisplayAll(w http.ResponseWriter, r *http.Request) {
	for _, v := range targetItemMap {
		displayData[v.JobName] = LinkLabel{v.Link.Link}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(displayData)
}

// Exposes the scrape targets on the appropriate end points
// If there is > 0 query params under the key 'collector_id' then it will only expose targets for that collector.
// Otherwise it will jsut expose all collector's jobs
func DisplayCollectorMapping(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()["collector_id"]

	var compareMap = make(map[string][]TargetItem)
	for _, v := range targetItemMap {
		compareMap[v.CollectorPtr.name+v.JobName] = append(compareMap[v.CollectorPtr.name+v.JobName], *v)
	}

	if len(q) == 0 {
		params := mux.Vars(r)
		for _, v := range targetItemMap {
			if v.JobName == params["job_id"] {
				for k := range displayData {
					delete(displayData, k)
				}
				var jobsArr []TargetItem
				jobsArr = append(jobsArr, compareMap[v.CollectorPtr.name+v.JobName]...)

				var targetGroupList []lbdiscovery.TargetGroup
				for _, v := range jobsArr {
					targetGroupList = append(targetGroupList, lbdiscovery.TargetGroup{Targets: []string{v.TargetUrl}, Labels: v.Label})

				}

				displayData2[v.CollectorPtr.name] = CollectorJson{Link: "/jobs/" + v.JobName + "/targets" + "?collector_id=" + v.CollectorPtr.name, Jobs: targetGroupList}

			}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(displayData2)

	} else {
		var tgs []lbdiscovery.TargetGroup
		for _, v := range colMap {
			if v.name == q[0] {
				for _, targetItemArr := range compareMap {
					for _, targetItem := range targetItemArr {
						if targetItem.CollectorPtr.name == q[0] {
							tgs = append(tgs, lbdiscovery.TargetGroup{Targets: []string{targetItem.TargetUrl}, Labels: targetItem.Label})
						}
					}
				}
			}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tgs)
	}
}

// Basic implementation of least connection algorithm - can be enhance or replaced by another delegation algorithm
func SetNextCollector() {
	for _, v := range colMap {
		if v.numTargs < next.nextCollector.numTargs {
			next.nextCollector = v
		}
	}
}

// Initlialize the set of targets which will be used to compare the targets in use by the collector instances
// This function will periodically be called when changes are made in the target discovery
func UpdateTargetSet() {
	for _, i := range targetList {
		targetSet[i.JobName+i.Target] = i
	}
}

// Initalize our set of collectors with key=collectorName, value=Collector object
// Collector instances are stable. Once initiated & allocated, these should not change. Only their jobs will change
func InitializeCollectors() {
	for _, i := range collectors {
		collector := Collector{name: i, numTargs: 0}
		colMap[i] = &collector
	}
	next.nextCollector = colMap[collectors[0]]
}

//Remove jobs from our struct that are no longer in the new set
func RemoveOutdatedJobs() {
	for k := range targetMap {
		if _, ok := targetSet[k]; !ok {
			delete(targetMap, k)
			colMap[targetItemMap[k].CollectorPtr.name].numTargs--
			delete(targetItemMap, k)
		}
	}
}

//Add jobs that were added into our struct
func AddUpdatedJobs() {
	for k, v := range targetSet {
		if _, ok := targetItemMap[k]; !ok {
			SetNextCollector()
			targetItem := TargetItem{JobName: v.JobName, Link: LinkLabel{"/jobs/" + v.JobName + "/targets"}, TargetUrl: v.Target, Label: v.Labels, CollectorPtr: next.nextCollector}
			next.nextCollector.numTargs++
			targetItemMap[v.JobName+v.Target] = &targetItem
		}
	}
}
