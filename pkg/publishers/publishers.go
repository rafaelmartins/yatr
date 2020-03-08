package publishers

import (
	"fmt"
	"os"
	"strings"

	"github.com/rafaelmartins/yatr/pkg/runners"
)

type Publisher interface {
	Name() string
	Detect(ctx *runners.Ctx) bool
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
			if ref := os.Getenv("GITHUB_REF"); ref != "refs/heads/master" && !strings.HasPrefix(ref, "refs/tags/") {
				return nil, fmt.Errorf("disabled, not master branch nor a git tag")
			}
		} else {
			return nil, fmt.Errorf("disabled, not push nor create event")
		}
	}

	for _, v := range publishers {
		if v.Detect(ctx) {
			return v, nil
		}
	}

	return nil, fmt.Errorf("disabled, no publisher available")
}
