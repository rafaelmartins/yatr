package publishers

import (
	"fmt"
	"os"
	"strings"

	"github.com/rafaelmartins/yatr/internal/publishers/distfiles_api"
	"github.com/rafaelmartins/yatr/internal/types"
)

type Publisher interface {
	Name() string
	Detect(ctx *types.Ctx) bool
	SetRelease(release bool)
	Publish(ctx *types.Ctx, proj *types.Project, archives []string, pattern string) error
}

var publishers = []Publisher{
	&distfiles_api.DistfilesApiPublisher{},
}

func Get(ctx *types.Ctx) (Publisher, error) {

	// publisher disabled by the user
	if v := strings.ToLower(os.Getenv("DISABLE_PUBLISHER")); v == "1" || v == "true" || v == "on" {
		return nil, fmt.Errorf("disabled by DISABLE_PUBLISHER")
	}

	// environment checks:
	// - don't run on pull requests
	// - only run for master and tags

	isRelease := false

	// travis
	if os.Getenv("TRAVIS") == "true" {
		if os.Getenv("TRAVIS_PULL_REQUEST") != "false" {
			return nil, fmt.Errorf("disabled, pull request")
		}
		if os.Getenv("TRAVIS_BRANCH") != "master" && os.Getenv("TRAVIS_TAG") == "" {
			return nil, fmt.Errorf("disabled, not master branch nor a git tag")
		}
		if os.Getenv("TRAVIS_TAG") != "" {
			isRelease = true
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
		if strings.HasPrefix(os.Getenv("GITHUB_REF"), "refs/tags/") {
			isRelease = true
		}
	}

	for _, v := range publishers {
		if v.Detect(ctx) {
			v.SetRelease(isRelease)
			return v, nil
		}
	}

	return nil, fmt.Errorf("disabled, no publisher available")
}
