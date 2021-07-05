package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"

	"github.com/http-sd-loadbalancer/targetdiscovery"

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
	numJobs int
	jobs    map[string]struct{}
	name    string
	jobName string
	link    string
}

// Label to display on the http server
type LinkLabel struct {
	Link string `json:"_link"`
}

type Targetgroup struct {
	Targets []string
	Labels  model.LabelSet
}
type CollectorJson struct {
	Link string      `json:"_link"`
	Jobs Targetgroup `json:"targets"`
}

// autoInc creates increasing id signatures: Ex -> 0, 1, 2, 3 when used 4 times
type autoInc struct {
	sync.Mutex
	id int
}

// belongs to autoInc
func (a *autoInc) ID() (id int) {
	a.Lock()
	defer a.Unlock()
	id = a.id
	a.id++
	return
}

var ai autoInc

// Next will hold the next collector pointer to be used when adding a new job (Uses least connection to be determined)
type Next struct {
	nextCollector *Collector
}

var next = Next{}

// -------------------------- Mock Data ------------------------------------
// Mock list of targets
var targets = []string{"localhost:0001", "localhost:0002", "localhost:0003", "localhost:0004",
	"localhost:0005", "localhost:0006", "localhost:0007", "localhost:0008", "localhost:0009", "localhost:0010",
	"localhost:0011", "localhost:0012", "localhost:0013", "localhost:0014", "localhost:0015", "localhost:0016", "localhost:0017"}

// Mock list of collector pod names
var collectors = []string{"collector-1", "collector-2", "collector-3", "collector-4"}

// -------------------------- End Mock Data --------------------------------

var targMap = make(map[int]targetdiscovery.TargetMapping)

var targetSet = make(map[string]struct{}) //set of targets - periodically updated // Once configured it will be updated with service discovery

var targetMap = make(map[string]string) //key=target, value=collectorName

var collectorMap = make(map[string]*Collector) // key=collectorName, value=Collector{}

var displayData = make(map[string]LinkLabel) // This is for the DisplayAll func

var displayData2 = make(map[string]CollectorJson) // This is for the DisplayCollectorMapping func

func main() {

	// TODO: Use service discovery instead of mock data && reformat structs for better performance / cleaner code
	tars, err := targetdiscovery.TargetDiscovery()
	if err != nil {
		fmt.Println(err)
	}

	for idx, v := range tars {
		targMap[idx] = v
	}

	for idx, v := range targMap {

		fmt.Println(idx, v)

	}
	// InitializeCollectors()
	// UpdateTargetSet()

	// // The following 2 function calls will reconcile the scrape targets.
	// // RemoveOutdatedJobs will compare internal map to the dynamically changing targetMap to remove any targets that are no longer being used
	// // AddUpdatedJobs will compare internal map to the dynamically changing targetMap to add any new targets
	// RemoveOutdatedJobs()
	// AddUpdatedJobs()

	// router := mux.NewRouter()
	// router.HandleFunc("/jobs", DisplayAll).Methods("GET")
	// router.HandleFunc("/jobs/{job_id}/targets", DisplayCollectorMapping).Methods("GET")
	// http.ListenAndServe(":3030", router)
}

func DisplayAll(w http.ResponseWriter, r *http.Request) {
	for _, v := range collectorMap {
		displayData[v.jobName] = LinkLabel{v.link}
	}
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
		for _, v := range collectorMap {
			if v.jobName == params["job_id"] {
				var jobsArr []string
				for i := range v.jobs {
					jobsArr = append(jobsArr, i)
				}
				displayData2[v.name] = CollectorJson{Link: "/jobs/" + v.jobName + "/targets" + "?collector_id=" + v.name, Jobs: Targetgroup{Targets: jobsArr, Labels: model.LabelSet{
					"__meta_label_datacenter": "dc3",
				}}}

			}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(displayData2)

	} else {
		var tgs []Targetgroup

		for _, v := range collectorMap {
			if v.name == q[0] {
				var jobsArr []string
				for i := range v.jobs {
					jobsArr = append(jobsArr, i)
				}
				tgs = []Targetgroup{
					{
						Targets: jobsArr,
						Labels: model.LabelSet{
							"__meta_label_datacenter": "dc3",
						},
					},
				}
			}
		}
		fmt.Print(q)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tgs)
	}
}

// Basic implementation of least connection algorithm - can be enhance or replaced by another delegation algorithm
func SetNextCollector() {
	for _, v := range collectorMap {
		if v.numJobs < next.nextCollector.numJobs {
			next.nextCollector = v
		}
	}
}

// AddJob adds new jobs to the next pointed collector (Currently using least connection)
func (c *Collector) AddJob(job string) {
	SetNextCollector()
	var exists = struct{}{}
	c.jobs[job] = exists
	c.numJobs++

}

// RemoveJob removes specified job
func (c *Collector) RemoveJob(job string) {
	if _, ok := c.jobs[job]; ok {
		delete(c.jobs, job)
		c.numJobs--
	}
}

// Initlialize the set of targets which will be used to compare the targets in use by the collector instances
// This function will periodically be called when changes are made in the target discovery
func UpdateTargetSet() {
	var exists = struct{}{}
	for _, i := range targets {
		targetSet[i] = exists
	}
}

// Initalize our set of collectors with key=collectorName, value=Collector object
// Collector instances are stable. Once initiated & allocated, these should not change. Only their jobs will change
func InitializeCollectors() {
	for range collectors {
		id := strconv.Itoa(ai.ID())
		tempJobName := "job" + id
		tempCollectorName := "collector-" + id
		coll := Collector{
			numJobs: 0,
			jobs:    make(map[string]struct{}),
			name:    tempCollectorName,
			jobName: tempJobName,
			link:    "/jobs/" + tempJobName + "/targets",
		}
		collectorMap[coll.name] = &coll
	}
	next.nextCollector = collectorMap[collectors[0]]
}

//Remove jobs from our struct that are no longer in the new set
func RemoveOutdatedJobs() {
	for k, v := range targetMap {
		if _, ok := targetSet[k]; !ok {
			delete(targetMap, k)
			collectorMap[v].RemoveJob(k)
		}
	}
}

//Add jobs that were added into our struct
func AddUpdatedJobs() {
	for i := range targetSet {
		for _, v := range collectorMap {
			if _, ok := v.jobs[i]; !ok {
				next.nextCollector.AddJob(i)
				break
			}
		}
	}
}
