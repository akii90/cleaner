package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type PolicyConfig struct {
	HealthyStatus     []string `yaml:"healthyStatus"`
	ExcludeNamespaces []string `yaml:"excludeNamespaces"`
	CheckDelaySeconds int      `yaml:"checkDelaySeconds"`
}

// LoadConfig reads the configuration file from given path
func LoadConfig(path string) (*PolicyConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var conf PolicyConfig
	if err := yaml.Unmarshal(data, &conf); err != nil {
		return nil, err
	}

	return &conf, nil
}
