package config

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
	ConfigureArgs        []string `yaml:"configure_args"`
	TaskArgs             []string `yaml:"task_args"`
	TaskScript           string   `yaml:"task_script"`
	ArchiveFilter        string   `yaml:"archive_filter"`
	ArchiveExtractFilter string   `yaml:"archive_extract_filter"`
	PublishOnFailure     bool     `yaml:"publish_on_failure"`
}

func Read(filename string) (*Config, error) {
	var conf Config

	configContent, err := ioutil.ReadFile(filename)
	if err != nil {
		return &conf, nil
	}

	if err := yaml.Unmarshal(configContent, &conf); err != nil {
		return nil, err
	}

	return &conf, nil
}
