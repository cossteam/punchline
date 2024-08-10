package config

import (
	"github.com/mitchellh/mapstructure"
	"gopkg.in/yaml.v3"
	"io/ioutil"
)

type Config struct {
	Subscriptions []struct {
		Topic string `yaml:"topic"`
	} `yaml:"subscriptions"`

	EndpointPort uint   `yaml:"endpointPort"`
	SignalServer string `yaml:"signalServer"`
	Server       string `yaml:"server"`
	Addr         string `yaml:"addr"`
	Hostname     string `yaml:"hostname"`

	StunServer string `yaml:"stunServer"`

	Logging struct {
		Level string `yaml:"level"`
	} `yaml:"logging"`

	Plugins []Plugin `yaml:"plugins"`
}

type Plugin struct {
	Name    string                 `yaml:"name"`
	Address string                 `yaml:"address"`
	Spec    map[string]interface{} `yaml:"spec"`
}

// LoadPluginConfig loads the specific configuration for a plugin
func (p *Plugin) LoadPluginConfig(target interface{}) error {
	return mapstructure.Decode(p.Spec, target)
}

func Load(filename string) (*Config, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err = yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
