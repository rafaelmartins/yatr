package main

import (
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type Config struct {
	DefaultConfigureArgs []string          `yaml:"default_configure_args"`
	DefaultTaskArgs      []string          `yaml:"default_task_args"`
	Targets              map[string]Target `yaml:"targets"`
}

type Target struct {
	ConfigureArgs []string `yaml:"configure_args"`
	TaskArgs      []string `yaml:"task_args"`
}

func ConfigRead(filename string) (*Config, error) {
	configContent, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := yaml.Unmarshal(configContent, &config); err != nil {
		return nil, err
	}
	return &config, nil
}
