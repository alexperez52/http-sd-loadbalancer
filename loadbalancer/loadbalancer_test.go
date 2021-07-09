package loadbalancer_test

import (
	"testing"

	"github.com/http-sd-loadbalancer/loadbalancer"
	"github.com/stretchr/testify/assert"
)

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
