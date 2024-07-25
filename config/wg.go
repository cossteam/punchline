package config

type WgSpec struct {
	Interfaces []Interface `yaml:"interfaces"`
}

type Interface struct {
	Iface    string   `yaml:"iface"`
	Hostname string   `yaml:"hostname"`
	Port     int      `yaml:"port"`
	Concern  []string `yaml:"concern"`
}
