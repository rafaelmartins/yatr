package main

import (
	"os"
)

type publisher interface {
	name() string
	publish(rctx *runnerCtx, bctx *buildCtx, pattern string) error
}

func getPublisher() publisher {

	// environment checks:
	// - don't run on pull requests
	// - only run for master and tags
	if os.Getenv("TRAVIS_PULL_REQUEST") != "false" {
		return nil
	}
	if os.Getenv("TRAVIS_BRANCH") != "master" && len(os.Getenv("TRAVIS_TAG")) == 0 {
		return nil
	}

	// distfiles api check
	distfilesApiUrl, found := os.LookupEnv("DISTFILES_URL")
	if found {
		return &distfilesApiPublisher{url: distfilesApiUrl}
	}

	return nil
}

func checkEnvPublisher() error {
	return nil
}
