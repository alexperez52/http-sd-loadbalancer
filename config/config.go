package config

import (
	"errors"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

var (
	// ErrInvalidLBYAML represents an error in the format of the original YAML configuration file.
	ErrInvalidLBYAML = errors.New("couldn't parse the loadbalancer configuration")
	// ErrInvalidLBFile represents an error in reading the original YAML configuration file.
	ErrInvalidLBFile = errors.New("couldn't read the loadbalancer configuration file")
)

var (
	defaultConfigFile string = "./conf/loadbalancer.yaml"
)

type Config struct {
	Mode          string            `yaml:"mode"`
	LabelSelector map[string]string `yaml:"label_selector,omitempty"`
	Config        ScrapeConfig      `yaml:"config"`
}

type ScrapeConfig struct {
	ScrapeConfigs []map[string]interface{} `yaml:"scrape_configs"`
}

func unmarshall(cfg *Config, configFile string) error {
	yamlFile, err := ioutil.ReadFile(configFile)
	if err != nil {
		return ErrInvalidLBFile
	}

	err = yaml.UnmarshalStrict(yamlFile, cfg)
	if err != nil {
		return ErrInvalidLBYAML
	}
	return nil
}

func Load(newConfigFile ...string) (Config, error) {
	cfg := Config{}
	configFile := defaultConfigFile
	if len(newConfigFile) > 0 {
		configFile = newConfigFile[0]
	}

	if err := unmarshall(&cfg, configFile); err != nil {
		return Config{}, err
	}

	return cfg, nil
}
