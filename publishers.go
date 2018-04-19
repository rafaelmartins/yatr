package main

import (
	"fmt"
	"os"
)

type publisher interface {
	name() string
	publish(rctx runnerCtx, bctx buildCtx, pattern string) error
}

func getPublisher() (publisher, error) {

	// publisher disabled by the user
	if _, found := os.LookupEnv("DISABLE_PUBLISHER"); found {
		return nil, fmt.Errorf("disabled by DISABLE_PUBLISHER")
	}

	// environment checks:
	// - don't run on pull requests
	// - only run for master and tags
	if os.Getenv("TRAVIS_PULL_REQUEST") != "false" {
		return nil, fmt.Errorf("disabled, pull request")
	}
	if os.Getenv("TRAVIS_BRANCH") != "master" && len(os.Getenv("TRAVIS_TAG")) == 0 {
		return nil, fmt.Errorf("disabled, not master branch nor a git tag")
	}

	// distfiles api check
	distfilesApiUrl, found := os.LookupEnv("DISTFILES_URL")
	if found {
		return &distfilesApiPublisher{url: distfilesApiUrl}, nil
	}

	return nil, fmt.Errorf("disabled, no publisher available")
}
