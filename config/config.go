package config

import (
	"errors"
	"fmt"
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
	defaultConfigFile string = "test.yaml"
)

type Config struct {
	Mode          string            `yaml:"mode"`
	LabelSelector map[string]string `yaml:"label_selector,omitempty"`
	Config        ScrapeConfig      `yaml:"config"`
}

type ScrapeConfig struct {
	ScrapeConfigs []map[string]interface{} `yaml:"scrape_configs"`
}

func unmarshall(cfg *Config) error {
	yamlFile, err := ioutil.ReadFile(defaultConfigFile)
	if err != nil {
		return ErrInvalidLBFile
	}

	err = yaml.UnmarshalStrict(yamlFile, cfg)
	if err != nil {
		return ErrInvalidLBYAML
	}
	return nil
}

func Load() Config {
	cfg := Config{}

	if err := unmarshall(&cfg); err != nil {
		fmt.Println(err)
	}

	return cfg
}
