package publishers

import (
	"fmt"
	"os"
	"strings"

	"github.com/rafaelmartins/yatr/pkg/runners"
)

type Publisher interface {
	Name() string
	Detect(ctx *runners.Ctx) Publisher
	Publish(ctx *runners.Ctx, proj *runners.Project, archives []string, pattern string) error
}

var publishers = []Publisher{
	&distfilesApiPublisher{},
}

func Get(ctx *runners.Ctx) (Publisher, error) {

	// publisher disabled by the user
	if _, found := os.LookupEnv("DISABLE_PUBLISHER"); found {
		return nil, fmt.Errorf("disabled by DISABLE_PUBLISHER")
	}

	// environment checks:
	// - don't run on pull requests
	// - only run for master and tags

	// travis
	if os.Getenv("TRAVIS") == "true" {
		if os.Getenv("TRAVIS_PULL_REQUEST") != "false" {
			return nil, fmt.Errorf("disabled, pull request")
		}
		if os.Getenv("TRAVIS_BRANCH") != "master" && len(os.Getenv("TRAVIS_TAG")) == 0 {
			return nil, fmt.Errorf("disabled, not master branch nor a git tag")
		}
	}

	// github actions
	if event, found := os.LookupEnv("GITHUB_EVENT_NAME"); found {
		if event == "push" {
			if os.Getenv("GITHUB_REF") != "refs/heads/master" {
				return nil, fmt.Errorf("disabled, not master branch for push event")
			}
		} else if event == "create" {
			if !strings.HasPrefix(os.Getenv("GITHUB_REF"), "refs/tags/") {
				return nil, fmt.Errorf("disabled, not a tag for create event")
			}
		} else {
			return nil, fmt.Errorf("disabled, not push nor create event")
		}
	}

	for _, v := range publishers {
		if r := v.Detect(ctx); r != nil {
			return r, nil
		}
	}

	return nil, fmt.Errorf("disabled, no publisher available")
}
