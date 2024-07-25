package config

type WgSpec struct {
	Interfaces []Interface `yaml:"interfaces"`
}

type Interface struct {
	Iface     string   `yaml:"iface"`
	Publickey string   `yaml:"publickey"`
	Port      int      `yaml:"port"`
	Concern   []string `yaml:"concern"`
}
