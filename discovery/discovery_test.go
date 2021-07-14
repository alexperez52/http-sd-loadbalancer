package discovery

import (
	"context"
	"io/ioutil"
	"os"
	"sort"
	"testing"

	"github.com/http-sd-loadbalancer/config"
	"github.com/http-sd-loadbalancer/suite"
	"github.com/stretchr/testify/assert"
)

func copyFileHelper(src string, dst string) error {
	input, err := ioutil.ReadFile(src)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(dst, input, 0644)
	if err != nil {
		return err
	}
	return nil
}

func copyFile(t testing.TB, src string, dst string) {
	t.Helper()
	tmp := "../suite/tmp.json"
	err := ioutil.WriteFile(tmp, []byte(""), 0644)
	assert.NoError(t, err)

	err = copyFileHelper(src, tmp)
	assert.NoError(t, err)
	err = copyFileHelper(dst, src)
	assert.NoError(t, err)
	err = copyFileHelper(tmp, dst)
	assert.NoError(t, err)

	err = os.Remove(tmp)
	assert.NoError(t, err)
}

func TestTargetDiscovery(t *testing.T) {
	defaultConfigTestFile := suite.GetConfigTestFile()
	cfg, err := config.Load(defaultConfigTestFile)
	assert.NoError(t, err)
	discoveryManager := NewManager(context.Background())

	t.Run("should discover targets", func(t *testing.T) {
		targets, err := Get(discoveryManager, cfg)
		assert.NoError(t, err)

		actualTargets := []string{}
		expectedTargets := []string{"prom.domain:9001", "prom.domain:9002", "prom.domain:9003", "promfile.domain:1001", "promfile.domain:3000"}

		assert.Len(t, targets, 5)
		for _, targets := range targets {
			actualTargets = append(actualTargets, targets.Target)
		}

		sort.Strings(expectedTargets)
		sort.Strings(actualTargets)

		assert.Equal(t, expectedTargets, actualTargets)

	})

	t.Run("should update targets", func(t *testing.T) {
		targets, err := Get(discoveryManager, cfg)
		assert.NoError(t, err)

		actualTargets := []string{}
		expectedTargets := []string{"prom.domain:9001", "prom.domain:9002", "prom.domain:9003", "promfile.domain:1001", "promfile.domain:3000", "promfile.domain:4000"}

		copyFile(t, suite.GetFileSdTestInitialFile(), suite.GetFileSdTestModFile())

		Watch(discoveryManager, &targets)

		assert.Len(t, targets, 6)
		for _, targets := range targets {
			actualTargets = append(actualTargets, targets.Target)
		}

		sort.Strings(expectedTargets)
		sort.Strings(actualTargets)

		assert.Equal(t, expectedTargets, actualTargets)

		copyFile(t, suite.GetFileSdTestInitialFile(), suite.GetFileSdTestModFile())

	})
}
