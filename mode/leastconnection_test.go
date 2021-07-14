package mode_test

import (
	"testing"

	lbdiscovery "github.com/http-sd-loadbalancer/discovery"
	"github.com/http-sd-loadbalancer/loadbalancer"
	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
)

// Tests least connection - The expected collector after running SetNextCollector should be the collecter with the least amount of workload
func TestSettingNextCollector(t *testing.T) {
	// prepare
	lb := loadbalancer.Init()
	defaultCol := loadbalancer.Collector{Name: "default-col", NumTargs: 1}
	maxCol := loadbalancer.Collector{Name: "max-col", NumTargs: 2}
	leastCol := loadbalancer.Collector{Name: "least-col", NumTargs: 0}
	lb.CollectorMap[maxCol.Name] = &maxCol
	lb.CollectorMap[leastCol.Name] = &leastCol
	lb.NextCol.NextCollector = &defaultCol

	// test
	lb.SetNextCollector()

	// verify
	assert.Equal(t, "least-col", lb.NextCol.NextCollector.Name)
}

func TestInitializingCollectors(t *testing.T) {
	// prepare
	cols := []string{"col-1", "col-2", "col-3"}
	lb := loadbalancer.Init()

	// test
	lb.InitializeCollectors(cols)

	// verify
	assert.Equal(t, len(cols), len(lb.CollectorMap))
	for _, i := range cols {
		assert.True(t, (lb.CollectorMap[i] != nil))
	}
}

func TestAddingAndRemovingTargetFlow(t *testing.T) {
	// prepare lb with initial targets and collectors
	lb := loadbalancer.Init()
	cols := []string{"col-1", "col-2", "col-3"}
	initTargets := []string{"targ:1000", "targ:1001", "targ:1002", "targ:1003", "targ:1004", "targ:1005"}
	lb.InitializeCollectors(cols)
	var targetList []lbdiscovery.TargetData
	for _, i := range initTargets {
		targetList = append(targetList, lbdiscovery.TargetData{JobName: "sample-name", Target: i, Labels: model.LabelSet{}})
	}

	// test that targets and collectors are added properly
	lb.UpdateTargetSet(targetList)
	lb.RefreshJobs()

	// verify
	assert.True(t, len(lb.TargetMap) == 6)
	assert.True(t, len(lb.TargetItemMap) == 6)

	// prepare second round of targets
	tar := []string{"targ:1001", "targ:1002", "targ:1003", "targ:1004"}
	var tarL []lbdiscovery.TargetData
	for _, i := range tar {
		tarL = append(tarL, lbdiscovery.TargetData{JobName: "sample-name", Target: i, Labels: model.LabelSet{}})
	}

	// test that less targets are found - removed
	lb.UpdateTargetSet(tarL)
	lb.RefreshJobs()

	// verify
	assert.True(t, len(lb.TargetMap) == 4)
	assert.True(t, len(lb.TargetItemMap) == 4)

	// verify results map
	for _, i := range tar {
		_, ok := lb.TargetMap["sample-name"+i]
		assert.True(t, ok)
	}
}
