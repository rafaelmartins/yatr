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
	ConfigureArgs   []string          `yaml:"configure_args"`
	TaskArgs        []string          `yaml:"task_args"`
	PublisherParams map[string]string `yaml:"publisher_params"`
}

func configRead(filename string) (*Config, error) {
	configContent, err := ioutil.ReadFile(filename)
	if err != nil {
		return &Config{}, err
	}

	var conf Config
	if err := yaml.Unmarshal(configContent, &conf); err != nil {
		return &Config{}, err
	}
	return &conf, nil
}
