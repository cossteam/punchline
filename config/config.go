package config

import (
	"gopkg.in/yaml.v3"
	"io/ioutil"
)

type Config struct {
	Publisher struct {
		Name string `yaml:"name"`
		Addr string `yaml:"addr"`
	} `yaml:"publisher"`
	Subscriptions []struct {
		Topic string `yaml:"topic"`
	} `yaml:"subscriptions"`

	UdpPort      uint   `yaml:"udp_port"`
	GrpcPort     uint   `yaml:"grpc_port"`
	EndpointPort uint   `yaml:"endpoint_port"` // 端点端口
	Server       string `yaml:"server"`
	Loglevel     string `yaml:"loglevel"`
	Addr         string `yaml:"addr"`
	Hostname     string `yaml:"hostname"`
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
