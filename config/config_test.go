package config

import (
	"testing"

	"github.com/http-sd-loadbalancer/suite"
	"github.com/stretchr/testify/assert"
)

func TestConfigLoad(t *testing.T) {
	// prepare
	cfg := Config{}
	defaultConfigTestFile := suite.GetConfigTestFile()
	expectedFileSDConfig := map[interface{}]interface{}{"files": []interface{}{"../suite/file_sd_test.yaml"}}
	expectedStaticSDConfig := map[interface{}]interface{}{"targets": []interface{}{"prom.domain:9001", "prom.domain:9002", "prom.domain:9003"}}

	// test
	err := unmarshall(&cfg, defaultConfigTestFile)
	actualFileSDConfig := cfg.Config.ScrapeConfigs[0]["file_sd_configs"].([]interface{})[0]
	actulaStaticSDConfig := cfg.Config.ScrapeConfigs[0]["static_configs"].([]interface{})[0]

	//verify
	assert.NoError(t, err)
	assert.Equal(t, cfg.Mode, "LeastConnection")
	assert.Equal(t, cfg.LabelSelector["app.kubernetes.io/instance"], "default.test")
	assert.Equal(t, cfg.LabelSelector["app.kubernetes.io/managed-by"], "opentelemetry-operator")
	assert.Equal(t, cfg.Config.ScrapeConfigs[0]["job_name"], "prometheus")
	assert.Equal(t, expectedFileSDConfig, actualFileSDConfig)
	assert.Equal(t, expectedStaticSDConfig, actulaStaticSDConfig)
}
