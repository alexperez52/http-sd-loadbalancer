package mode

import (
	"log"

	lbdiscovery "github.com/http-sd-loadbalancer/discovery"
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
	Name     string
	NumTargs int
}

// Label to display on the http server
type LinkLabel struct {
	Link string `json:"_link"`
}

type CollectorJson struct {
	Link string                    `json:"_link"`
	Jobs []lbdiscovery.TargetGroup `json:"targets"`
}

// Next will hold the next collector pointer to be used when adding a new job (Uses least connection to be determined)
type Next struct {
	NextCollector *Collector
}

type TargetItem struct {
	JobName      string
	Link         LinkLabel
	TargetUrl    string
	Label        model.LabelSet
	CollectorPtr *Collector
}

type DisplayCache struct {
	DisplayJobs          map[string](map[string][]lbdiscovery.TargetGroup)
	DisplayCollectorJson map[string](map[string]CollectorJson)
	DisplayJobMapping    map[string]LinkLabel
	DisplayTargetMapping map[string][]lbdiscovery.TargetGroup
}

type LoadBalancer struct {
	TargetSet     map[string]lbdiscovery.TargetData
	TargetMap     map[string]lbdiscovery.TargetData
	CollectorMap  map[string]*Collector
	TargetItemMap map[string]*TargetItem
	Cache         DisplayCache
	NextCol       Next
}

// Basic implementation of least connection algorithm - can be enhance or replaced by another delegation algorithm
func (lb *LoadBalancer) SetNextCollector() {
	for _, v := range lb.CollectorMap {
		if v.NumTargs < lb.NextCol.NextCollector.NumTargs {
			lb.NextCol.NextCollector = v
		}
	}
}

// Initlialize the set of targets which will be used to compare the targets in use by the collector instances
// This function will periodically be called when changes are made in the target discovery
func (lb *LoadBalancer) UpdateTargetSet(targetList []lbdiscovery.TargetData) {
	// Dump old data
	for k := range lb.TargetSet {
		delete(lb.TargetSet, k)
	}

	// Set new data
	for _, i := range targetList {
		lb.TargetSet[i.JobName+i.Target] = i
	}
}

// Initalize our set of collectors with key=collectorName, value=Collector object
// Collector instances are stable. Once initiated & allocated, these should not change. Only their jobs will change
func (lb *LoadBalancer) InitializeCollectors(collectors []string) {
	if len(collectors) == 0 {
		log.Fatal("no collector instances present")
	}

	for _, i := range collectors {
		collector := Collector{Name: i, NumTargs: 0}
		lb.CollectorMap[i] = &collector
	}
	lb.NextCol.NextCollector = lb.CollectorMap[collectors[0]]
}

//Remove jobs from our struct that are no longer in the new set
func (lb *LoadBalancer) RemoveOutdatedTargets() {
	for k := range lb.TargetMap {
		if _, ok := lb.TargetSet[k]; !ok {
			delete(lb.TargetMap, k)
			lb.CollectorMap[lb.TargetItemMap[k].CollectorPtr.Name].NumTargs--
			delete(lb.TargetItemMap, k)
		}
	}
}

//Add jobs that were added into our struct
func (lb *LoadBalancer) AddUpdatedTargets() {
	for k, v := range lb.TargetSet {
		if _, ok := lb.TargetItemMap[k]; !ok {
			lb.SetNextCollector()
			lb.TargetMap[k] = v
			targetItem := TargetItem{JobName: v.JobName, Link: LinkLabel{"/jobs/" + v.JobName + "/targets"}, TargetUrl: v.Target, Label: v.Labels, CollectorPtr: lb.NextCol.NextCollector}
			lb.NextCol.NextCollector.NumTargs++
			lb.TargetItemMap[v.JobName+v.Target] = &targetItem
		}
	}
}

func (lb *LoadBalancer) GenerateCache() {
	var compareMap = make(map[string][]TargetItem) // CollectorName+jobName -> TargetItem
	for _, targetItem := range lb.TargetItemMap {
		compareMap[targetItem.CollectorPtr.Name+targetItem.JobName] = append(compareMap[targetItem.CollectorPtr.Name+targetItem.JobName], *targetItem)
	}
	lb.Cache = DisplayCache{DisplayJobs: make(map[string]map[string][]lbdiscovery.TargetGroup), DisplayCollectorJson: make(map[string](map[string]CollectorJson))}
	for _, v := range lb.TargetItemMap {
		lb.Cache.DisplayJobs[v.JobName] = make(map[string][]lbdiscovery.TargetGroup)
	}
	for _, v := range lb.TargetItemMap {
		var jobsArr []TargetItem
		jobsArr = append(jobsArr, compareMap[v.CollectorPtr.Name+v.JobName]...)

		var targetGroupList []lbdiscovery.TargetGroup
		targetItemSet := make(map[string][]TargetItem)
		for _, m := range jobsArr {
			targetItemSet[m.JobName+m.Label.String()] = append(targetItemSet[m.JobName+m.Label.String()], m)
		}
		labelSet := make(map[string]model.LabelSet)
		for _, targetItemList := range targetItemSet {
			var targetArr []string
			for _, targetItem := range targetItemList {
				labelSet[targetItem.TargetUrl] = targetItem.Label
				targetArr = append(targetArr, targetItem.TargetUrl)
			}
			targetGroupList = append(targetGroupList, lbdiscovery.TargetGroup{Targets: targetArr, Labels: labelSet[targetArr[0]]})

		}
		lb.Cache.DisplayJobs[v.JobName][v.CollectorPtr.Name] = targetGroupList
	}
}

// TODO: Add mutex
// UpdateCache gets called whenever RefreshJobs gets called
func (lb *LoadBalancer) UpdateCache() {
	lb.GenerateCache() // Create cached structure
	// Create the display maps
	lb.Cache.DisplayTargetMapping = make(map[string][]lbdiscovery.TargetGroup)
	lb.Cache.DisplayJobMapping = make(map[string]LinkLabel)
	for _, vv := range lb.TargetItemMap {
		lb.Cache.DisplayCollectorJson[vv.JobName] = make(map[string]CollectorJson)
	}
	for k, v := range lb.Cache.DisplayJobs {
		for kk, vv := range v {
			lb.Cache.DisplayCollectorJson[k][kk] = CollectorJson{Link: "/jobs/" + k + "/targets" + "?collector_id=" + kk, Jobs: vv}
		}
	}
	for _, targetItem := range lb.TargetItemMap {
		lb.Cache.DisplayJobMapping[targetItem.JobName] = LinkLabel{targetItem.Link.Link}
	}

	for k, v := range lb.Cache.DisplayJobs {
		for kk, vv := range v {
			lb.Cache.DisplayTargetMapping[k+kk] = vv
		}
	}

}

// TODO: Add boolean flags to determine if any changes were made that should trigger RefreshJobs
// RefreshJobs is a function that is called periodically - this will create a cached structure to hold data for consistency
// when collectors perform GET operations
func (lb *LoadBalancer) RefreshJobs() {
	lb.RemoveOutdatedTargets()
	lb.AddUpdatedTargets()
	lb.UpdateCache()
}

// UpdateCache updates the DisplayMap so that mapping is consistent

func Init() *LoadBalancer {
	lb := LoadBalancer{
		TargetSet:     make(map[string]lbdiscovery.TargetData),
		TargetMap:     make(map[string]lbdiscovery.TargetData),
		CollectorMap:  make(map[string]*Collector),
		TargetItemMap: make(map[string]*TargetItem),
		NextCol:       Next{}}
	return &lb
}
