package main

import (
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type config struct {
	DefaultConfigureArgs []string          `yaml:"default_configure_args"`
	DefaultTaskArgs      []string          `yaml:"default_task_args"`
	Targets              map[string]target `yaml:"targets"`
}

type target struct {
	ConfigureArgs        []string `yaml:"configure_args"`
	TaskArgs             []string `yaml:"task_args"`
	ArchiveFilter        string   `yaml:"archive_filter"`
	ArchiveExtractFilter string   `yaml:"archive_extract_filter"`
}

func configRead(filename string) (*config, error) {
	configContent, err := ioutil.ReadFile(filename)
	if err != nil {
		return &config{}, nil
	}

	var conf config
	if err := yaml.Unmarshal(configContent, &conf); err != nil {
		return &config{}, err
	}
	return &conf, nil
}
