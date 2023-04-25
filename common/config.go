package common

import (
	"io/ioutil"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Common BasicConfig `yaml:"common"`
	Log    LogConfig   `yaml:"log"`
	Cache  CacheConfig `yaml:"cache"`
}

type BasicConfig struct {
	Name  string `yaml:"name"`
	Queen string `yaml:"queen"`
	Proxy bool   `yaml:"proxy"`
	Cache bool   `yaml:"cache"`
}

type ProxyConfig struct {
	Addr string `yaml:"addr"`
}

type LogConfig struct {
	Path   string `yaml:"path"`
	Level  string `yaml:"level"`
	Access string `yaml:"access"`
}

type CacheConfig struct {
	Addr      string      `yaml:"addr"`
	AdminAddr string      `yaml:"admin"`
	Device    []DeviceCfg `yaml:"device"`
}

type DeviceCfg struct {
	Name string `yaml:"name"`
	Dir  string `yaml:"dir"`
	Size string `yaml:"size"`
}

// parseConfig parses the YAML config file and performs validation on the fields.
func parseConfig(filename string) (*Config, error) {
	data, err := ioutil.ReadFile(filename)
	Success(err)

	config := &Config{}
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, err
	}

	for _, device := range config.Cache.Device {
		ParseSize(device.Size)
	}

	return config, nil
}

func LoadConf(filePath string) Config {
	config, err := parseConfig(filePath)
	if err != nil {
		panic(err)
	}
	return *config
}
