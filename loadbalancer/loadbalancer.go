package loadbalancer

import (
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

type LoadBalancer struct {
	TargetSet     map[string]lbdiscovery.TargetData
	TargetMap     map[string]lbdiscovery.TargetData
	CollectorMap  map[string]*Collector
	DisplayData   map[string]LinkLabel
	DisplayData2  map[string]CollectorJson
	TargetItemMap map[string]*TargetItem
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
	for _, i := range targetList {
		lb.TargetSet[i.JobName+i.Target] = i
	}
}

// Initalize our set of collectors with key=collectorName, value=Collector object
// Collector instances are stable. Once initiated & allocated, these should not change. Only their jobs will change
func (lb *LoadBalancer) InitializeCollectors(collectors []string) {
	for _, i := range collectors {
		collector := Collector{Name: i, NumTargs: 0}
		lb.CollectorMap[i] = &collector
	}
	lb.NextCol.NextCollector = lb.CollectorMap[collectors[0]]
}

//Remove jobs from our struct that are no longer in the new set
func (lb *LoadBalancer) RemoveOutdatedJobs() {
	for k := range lb.TargetMap {
		if _, ok := lb.TargetSet[k]; !ok {
			delete(lb.TargetMap, k)
			lb.CollectorMap[lb.TargetItemMap[k].CollectorPtr.Name].NumTargs--
			delete(lb.TargetItemMap, k)
		}
	}
}

//Add jobs that were added into our struct
func (lb *LoadBalancer) AddUpdatedJobs() {
	for k, v := range lb.TargetSet {
		if _, ok := lb.TargetItemMap[k]; !ok {
			lb.SetNextCollector()
			targetItem := TargetItem{JobName: v.JobName, Link: LinkLabel{"/jobs/" + v.JobName + "/targets"}, TargetUrl: v.Target, Label: v.Labels, CollectorPtr: lb.NextCol.NextCollector}
			lb.NextCol.NextCollector.NumTargs++
			lb.TargetItemMap[v.JobName+v.Target] = &targetItem
		}
	}
}

func SetupDisplayTargets(lb *LoadBalancer, p map[string]string) map[string]CollectorJson {
	targets := make(map[string]CollectorJson)
	var compareMap = make(map[string][]TargetItem) // CollectorName+jobName -> TargetItem
	for _, targetItem := range lb.TargetItemMap {
		compareMap[targetItem.CollectorPtr.Name+targetItem.JobName] = append(compareMap[targetItem.CollectorPtr.Name+targetItem.JobName], *targetItem)
	}
	for i := range targets {
		delete(targets, i)
	}

	for _, job := range lb.TargetItemMap {
		if job.JobName == p["job_id"] {
			var jobsArr []TargetItem
			jobsArr = append(jobsArr, compareMap[job.CollectorPtr.Name+job.JobName]...)

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
			targets[job.CollectorPtr.Name] = CollectorJson{Link: "/jobs/" + job.JobName + "/targets" + "?collector_id=" + job.CollectorPtr.Name, Jobs: targetGroupList}
		}
	}
	return targets
}

func SetupCollectorData(lb *LoadBalancer, q []string) []lbdiscovery.TargetGroup {
	targets := []lbdiscovery.TargetGroup{}
	var compareMap = make(map[string][]TargetItem) // CollectorName+jobName -> TargetItem
	for _, targetItem := range lb.TargetItemMap {
		compareMap[targetItem.CollectorPtr.Name+targetItem.JobName] = append(compareMap[targetItem.CollectorPtr.Name+targetItem.JobName], *targetItem)
	}
	targetMap := make(map[string][]string)
	labelSet := make(map[string]model.LabelSet)
	for _, collector := range lb.CollectorMap {
		if collector.Name == q[0] {
			for _, targetItemArr := range compareMap {
				for _, targetItem := range targetItemArr {
					if targetItem.CollectorPtr.Name == q[0] {
						targetMap[targetItem.Label.String()] = append(targetMap[targetItem.Label.String()], targetItem.TargetUrl)
						labelSet[targetItem.TargetUrl] = targetItem.Label
					}
				}
			}
		}
	}

	for _, v := range targetMap {
		targets = append(targets, lbdiscovery.TargetGroup{Targets: v, Labels: labelSet[v[0]]})
	}
	return targets
}

func SetupDisplayData(lb *LoadBalancer) map[string]LinkLabel {
	displayData := make(map[string]LinkLabel)
	for _, targetItem := range lb.TargetItemMap {
		displayData[targetItem.JobName] = LinkLabel{targetItem.Link.Link}
	}
	return displayData
}

func (lb *LoadBalancer) RefreshJobs() {
	lb.RemoveOutdatedJobs()
	lb.AddUpdatedJobs()
}

func Init() *LoadBalancer {
	lb := LoadBalancer{TargetSet: make(map[string]lbdiscovery.TargetData),
		TargetMap:     make(map[string]lbdiscovery.TargetData),
		CollectorMap:  make(map[string]*Collector),
		DisplayData:   make(map[string]LinkLabel),
		DisplayData2:  make(map[string]CollectorJson),
		TargetItemMap: make(map[string]*TargetItem),
		NextCol:       Next{}}
	return &lb
}
