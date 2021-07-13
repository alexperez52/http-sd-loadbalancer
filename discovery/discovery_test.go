package discovery

import (
	"context"
	"testing"

	"github.com/http-sd-loadbalancer/config"
	"github.com/http-sd-loadbalancer/suite"
	"github.com/stretchr/testify/assert"
)

func TestTargetDiscovery(t *testing.T) {
	defaultConfigTestFile := suite.GetConfigTestFile()
	cfg, err := config.Load(defaultConfigTestFile)
	assert.NoError(t, err)
	discoveryManager := NewManager(context.Background())

	t.Run("should discover targets", func(t *testing.T) {
		targets, err := Get(discoveryManager, cfg)
		assert.NoError(t, err)

		actualTargets := []string{}
		expectedTargets := []string{"prom.domain:9001", "prom.domain:9002", "prom.domain:9003", "promfile.domain:1001", "promfile.domain:1002", "promfile.domain:1003", "promfile.domain:3000"}

		assert.Len(t, targets, 7)
		for _, targets := range targets {
			actualTargets = append(actualTargets, targets.Target)
		}
		assert.Equal(t, expectedTargets, actualTargets)

	})
}
