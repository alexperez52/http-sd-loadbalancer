package targetdiscovery

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/go-kit/log"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/discovery"
	"github.com/prometheus/prometheus/discovery/azure"
	"github.com/prometheus/prometheus/discovery/consul"
	"github.com/prometheus/prometheus/discovery/digitalocean"
	"github.com/prometheus/prometheus/discovery/dns"
	"github.com/prometheus/prometheus/discovery/eureka"
	"github.com/prometheus/prometheus/discovery/file"
	"github.com/prometheus/prometheus/discovery/gce"
	"github.com/prometheus/prometheus/discovery/hetzner"
	"github.com/prometheus/prometheus/discovery/http"
	"github.com/prometheus/prometheus/discovery/kubernetes"
	"github.com/prometheus/prometheus/discovery/linode"
	"github.com/prometheus/prometheus/discovery/marathon"
	"github.com/prometheus/prometheus/discovery/openstack"
	"github.com/prometheus/prometheus/discovery/scaleway"
	"github.com/prometheus/prometheus/discovery/triton"
	yaml "gopkg.in/yaml.v2"
)

var (
	mapping = map[string]interface{}{
		"file": file.SDConfig{},
	}
)

var (
	// ErrInvalidLBYAML represents an error in the format of the original YAML configuration file.
	ErrInvalidLBYAML = errors.New("couldn't parse the loadbalancer configuration")
	// ErrInvalidLBFile represents an error in reading the original YAML configuration file.
	ErrInvalidLBFile = errors.New("couldn't read the loadbalancer configuration file")
)

type Config struct {
	Mode          string       `yaml:"mode"`
	LabelSelector string       `yaml:"label_selector"`
	Config        ScrapeConfig `yaml:"config"`
}

type ScrapeConfig struct {
	ScrapeConfigs []map[string]interface{} `yaml:"scrape_configs"`
}

type TargetGroup struct {
	Targets []string
	Labels  model.LabelSet
}

type TargetMapping struct {
	JobName     string
	TargetGroup TargetGroup
}

func Run(discoveryManager *discovery.Manager) error {
	if err := discoveryManager.Run(); err != nil {
		return fmt.Errorf("Discovery manager failed")
	}
	return nil
}

func getTargets(discoveryManager *discovery.Manager) ([]TargetMapping, error) {
	tsets := <-discoveryManager.SyncCh()
	var targets = []TargetMapping{}

	for jobName, tgs := range tsets {
		var targetGroup = TargetGroup{}
		for _, target := range tgs {

			targetInterface, err := target.MarshalYAML()
			if err != nil {
				return nil, err
			}

			targetYAML, err := yaml.Marshal(targetInterface)
			if err != nil {
				return nil, err
			}

			if err = yaml.Unmarshal(targetYAML, &targetGroup); err != nil {
				return nil, err
			}

			targets = append(targets, TargetMapping{JobName: jobName, TargetGroup: targetGroup})
		}
	}
	return targets, nil
}

func unmarshall(filePath string, cfg *Config) error {
	yamlFile, err := ioutil.ReadFile(filePath)
	if err != nil {
		return ErrInvalidLBFile
	}

	err = yaml.UnmarshalStrict(yamlFile, cfg)
	if err != nil {
		return ErrInvalidLBYAML
	}
	return nil
}

func TargetDiscovery() ([]TargetMapping, error) {
	var cfg = Config{}

	if err := unmarshall("test.yaml", &cfg); err != nil {
		fmt.Println(err)
	}

	discoveryCtx, _ := context.WithCancel(context.Background())
	discoveryManager := discovery.NewManager(discoveryCtx, log.NewNopLogger())
	discoveryCfg := make(map[string]discovery.Configs)
	discoveryConfigs := discovery.Configs{}

	for _, scrapeConfig := range cfg.Config.ScrapeConfigs {
		for name, sd := range scrapeConfig {
			if strings.HasSuffix(name, "_sd_configs") {
				sdYAML, _ := yaml.Marshal(sd)
				switch name {
				case "azure_sd_configs":
					sdConfig := []azure.SDConfig{}
					err := yaml.UnmarshalStrict(sdYAML, &sdConfig)
					if err != nil {
						fmt.Printf("error unmarshalling azure sd config: %s", err)
					}
					for index, _ := range sdConfig {
						discoveryConfigs = append(discoveryConfigs, &sdConfig[index])
					}
				case "consul_sd_configs":
					sdConfig := []consul.SDConfig{}
					err := yaml.UnmarshalStrict(sdYAML, &sdConfig)
					if err != nil {
						fmt.Printf("error unmarshalling consul sd config: %s", err)
					}
					for index, _ := range sdConfig {
						discoveryConfigs = append(discoveryConfigs, &sdConfig[index])
					}
				case "digitalocean_sd_configs":
					sdConfig := []digitalocean.SDConfig{}
					err := yaml.UnmarshalStrict(sdYAML, &sdConfig)
					if err != nil {
						fmt.Printf("error unmarshalling digitalocean sd config: %s", err)
					}
					for index, _ := range sdConfig {
						discoveryConfigs = append(discoveryConfigs, &sdConfig[index])
					}
				// case "docker_sd_configs":
				// 	sdConfig := []docker.SDConfig{}
				// 	err := yaml.UnmarshalStrict(sdYAML, &sdConfig)
				// 	if err != nil {
				// 		fmt.Printf("error unmarshalling docker sd config: %s", err)
				// 	}
				// 	for index, _ := range sdConfig {
				// 		discoveryConfigs = append(discoveryConfigs, &sdConfig[index])
				// 	}
				// case "dockerswarm_sd_configs":
				// 	sdConfig := []dockerswarm.SDConfig{}
				// 	err := yaml.UnmarshalStrict(sdYAML, &sdConfig)
				// 	if err != nil {
				// 		fmt.Printf("error unmarshalling dockerswarm sd config: %s", err)
				// 	}
				// 	for index, _ := range sdConfig {
				// 		discoveryConfigs = append(discoveryConfigs, &sdConfig[index])
				// 	}
				case "dns_sd_configs":
					sdConfig := []dns.SDConfig{}
					err := yaml.UnmarshalStrict(sdYAML, &sdConfig)
					if err != nil {
						fmt.Printf("error unmarshalling dns sd config: %s", err)
					}
					for index, _ := range sdConfig {
						discoveryConfigs = append(discoveryConfigs, &sdConfig[index])
					}
				// case "ec2_sd_configs":
				// 	sdConfig := []ec2.SDConfig{}
				// 	err := yaml.UnmarshalStrict(sdYAML, &sdConfig)
				// 	if err != nil {
				// 		fmt.Printf("error unmarshalling ec2 sd config: %s", err)
				// 	}
				// 	for index, _ := range sdConfig {
				// 		discoveryConfigs = append(discoveryConfigs, &sdConfig[index])
				// 	}
				case "openstack_sd_configs":
					sdConfig := []openstack.SDConfig{}
					err := yaml.UnmarshalStrict(sdYAML, &sdConfig)
					if err != nil {
						fmt.Printf("error unmarshalling openstack sd config: %s", err)
					}
					for index, _ := range sdConfig {
						discoveryConfigs = append(discoveryConfigs, &sdConfig[index])
					}
				case "file_sd_configs":
					sdConfig := []file.SDConfig{}
					err := yaml.UnmarshalStrict(sdYAML, &sdConfig)
					if err != nil {
						fmt.Printf("error unmarshalling file sd config: %s", err)
					}
					for index, _ := range sdConfig {
						discoveryConfigs = append(discoveryConfigs, &sdConfig[index])
					}
				case "gce_sd_configs":
					sdConfig := []gce.SDConfig{}
					err := yaml.UnmarshalStrict(sdYAML, &sdConfig)
					if err != nil {
						fmt.Printf("error unmarshalling gce sd config: %s", err)
					}
					for index, _ := range sdConfig {
						discoveryConfigs = append(discoveryConfigs, &sdConfig[index])
					}
				case "hetzner_sd_configs":
					sdConfig := []hetzner.SDConfig{}
					err := yaml.UnmarshalStrict(sdYAML, &sdConfig)
					if err != nil {
						fmt.Printf("error unmarshalling hetzner sd config: %s", err)
					}
					for index, _ := range sdConfig {
						discoveryConfigs = append(discoveryConfigs, &sdConfig[index])
					}
				case "http_sd_configs":
					sdConfig := []http.SDConfig{}
					err := yaml.UnmarshalStrict(sdYAML, &sdConfig)
					if err != nil {
						fmt.Printf("error unmarshalling http sd config: %s", err)
					}
					for index, _ := range sdConfig {
						discoveryConfigs = append(discoveryConfigs, &sdConfig[index])
					}
				case "kubernetes_sd_configs":
					sdConfig := []kubernetes.SDConfig{}
					err := yaml.UnmarshalStrict(sdYAML, &sdConfig)
					if err != nil {
						fmt.Printf("error unmarshalling kubernetes sd config: %s", err)
					}
					for index, _ := range sdConfig {
						discoveryConfigs = append(discoveryConfigs, &sdConfig[index])
					}
				// case "lightsail_sd_configs":
				// 	sdConfig := []lightsail.SDConfig{}
				// 	err := yaml.UnmarshalStrict(sdYAML, &sdConfig)
				// 	if err != nil {
				// 		fmt.Printf("error unmarshalling lightsail sd config: %s", err)
				// 	}
				// 	for index, _ := range sdConfig {
				// 		discoveryConfigs = append(discoveryConfigs, &sdConfig[index])
				// 	}
				case "linode_sd_configs":
					sdConfig := []linode.SDConfig{}
					err := yaml.UnmarshalStrict(sdYAML, &sdConfig)
					if err != nil {
						fmt.Printf("error unmarshalling linode sd config: %s", err)
					}
					for index, _ := range sdConfig {
						discoveryConfigs = append(discoveryConfigs, &sdConfig[index])
					}
				case "marathon_sd_configs":
					sdConfig := []marathon.SDConfig{}
					err := yaml.UnmarshalStrict(sdYAML, &sdConfig)
					if err != nil {
						fmt.Printf("error unmarshalling marathon sd config: %s", err)
					}
					for index, _ := range sdConfig {
						discoveryConfigs = append(discoveryConfigs, &sdConfig[index])
					}
				// case "nerve_sd_configs":
				// 	sdConfig := []nerve.SDConfig{}
				// 	err := yaml.UnmarshalStrict(sdYAML, &sdConfig)
				// 	if err != nil {
				// 		fmt.Printf("error nerve azure sd config: %s", err)
				// 	}
				// 	for index, _ := range sdConfig {
				// 		discoveryConfigs = append(discoveryConfigs, &sdConfig[index])
				// 	}
				// case "serverset_sd_configs":
				// 	sdConfig := []serverset.SDConfig{}
				// 	err := yaml.UnmarshalStrict(sdYAML, &sdConfig)
				// 	if err != nil {
				// 		fmt.Printf("error serverset azure sd config: %s", err)
				// 	}
				// 	for index, _ := range sdConfig {
				// 		discoveryConfigs = append(discoveryConfigs, &sdConfig[index])
				// 	}
				case "triton_sd_configs":
					sdConfig := []triton.SDConfig{}
					err := yaml.UnmarshalStrict(sdYAML, &sdConfig)
					if err != nil {
						fmt.Printf("error unmarshalling triton sd config: %s", err)
					}
					for index, _ := range sdConfig {
						discoveryConfigs = append(discoveryConfigs, &sdConfig[index])
					}
				case "eureka_sd_configs":
					sdConfig := []eureka.SDConfig{}
					err := yaml.UnmarshalStrict(sdYAML, &sdConfig)
					if err != nil {
						fmt.Printf("error unmarshalling eureka sd config: %s", err)
					}
					for index, _ := range sdConfig {
						discoveryConfigs = append(discoveryConfigs, &sdConfig[index])
					}
				case "scaleway_sd_configs":
					sdConfig := []scaleway.SDConfig{}
					err := yaml.UnmarshalStrict(sdYAML, &sdConfig)
					if err != nil {
						fmt.Printf("error unmarshalling scaleway sd config: %s", err)
					}
					for index, _ := range sdConfig {
						discoveryConfigs = append(discoveryConfigs, &sdConfig[index])
					}
				}
			} else if name == "static_configs" {
				staticYAML, _ := yaml.Marshal(sd)
				staticConfig := discovery.StaticConfig{}
				err := yaml.UnmarshalStrict(staticYAML, &staticConfig)
				if err != nil {
					fmt.Printf("error unmarshalling static config: %s", err)
				}
				discoveryConfigs = append(discoveryConfigs, staticConfig)
			}
		}
		discoveryCfg[scrapeConfig["job_name"].(string)] = discoveryConfigs
	}

	if err := discoveryManager.ApplyConfig(discoveryCfg); err != nil {
		return nil, err
	}

	go Run(discoveryManager)

	targets, err := getTargets(discoveryManager)
	if err != nil {
		return nil, err
	}

	return targets, nil
}