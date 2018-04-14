package main

import (
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type config struct {
	defaultConfigureArgs []string          `yaml:"default_configure_args"`
	defaultTaskArgs      []string          `yaml:"default_task_args"`
	targets              map[string]target `yaml:"targets"`
}

type target struct {
	configureArgs   []string          `yaml:"configure_args"`
	taskArgs        []string          `yaml:"task_args"`
	publisherParams map[string]string `yaml:"publisher_params"`
}

func configRead(filename string) (*config, error) {
	configContent, err := ioutil.ReadFile(filename)
	if err != nil {
		return &config{}, err
	}

	var conf config
	if err := yaml.Unmarshal(configContent, &conf); err != nil {
		return &config{}, err
	}
	return &conf, nil
}
