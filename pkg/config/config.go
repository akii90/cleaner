package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type PolicyConfig struct {
	ExcludePodStatus  []string `yaml:"excludePodStatus"`
	ExcludeNamespaces []string `yaml:"excludeNamespaces"`
	CheckDelaySeconds int      `yaml:"checkDelaySeconds"`
}

// LoadConfig reads the configuration file from given path
// Use default if path is not given
func LoadConfig(path string) (*PolicyConfig, error) {
	var conf PolicyConfig
	if path == "" {
		return defaultConfig(), nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	if err := yaml.Unmarshal(data, &conf); err != nil {
		return nil, err
	}

	return &conf, nil
}

// defaultConfig
func defaultConfig() *PolicyConfig {
	return &PolicyConfig{
		ExcludePodStatus:  []string{"Running", "Init"},
		ExcludeNamespaces: []string{"kube-system"},
		CheckDelaySeconds: 180,
	}
}
