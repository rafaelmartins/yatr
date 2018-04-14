package main

import (
	"fmt"
	"log"
	"os"
)

type publisher interface {
	publish(rctx *runnerCtx, bctx *buildCtx, params map[string]string) error
}

func getPublisher() (publisher, error) {

	// environment checks:
	// - don't run on pull requests
	// - only run for master and tags
	if os.Getenv("TRAVIS_PULL_REQUEST") != "false" {
		log.Println("warning: this seems to be a pull request. not publishing ...")
		return nil, nil
	}
	if os.Getenv("TRAVIS_BRANCH") != "master" && len(os.Getenv("TRAVIS_TAG")) == 0 {
		log.Println("warning: this seems to not be master branch nor a git tag. not publishing ...")
		return nil, nil
	}

	// distfiles api check
	distfilesApiUrl, found := os.LookupEnv("DISTFILES_URL")
	if found {
		return &distfilesApiPublisher{url: distfilesApiUrl}, nil
	}

	return nil, fmt.Errorf("error: No publisher found!")
}

func checkEnvPublisher() error {
	return nil
}
