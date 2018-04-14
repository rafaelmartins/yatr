package main

import (
	"log"
	"os"
	"path"
)

func main() {
	targetName, ok := os.LookupEnv("TARGET")
	if !ok {
		log.Fatalln("error: Target not provided")
	}

	if os.Getenv("TRAVIS") != "true" {
		log.Fatalln("error: This tool only supports Travis-CI")
	}

	conf, _ := configRead(".yatr.yml")

	target := conf.targets[targetName]

	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	run, rctx, err := getRunner(targetName, dir, path.Join(dir, "build"))
	if err != nil {
		log.Fatal(err)
	}

	configureArgs := append(conf.defaultConfigureArgs, target.configureArgs...)
	if err := run.configure(rctx, configureArgs); err != nil {
		log.Fatal(err)
	}

	taskArgs := append(conf.defaultTaskArgs, target.taskArgs...)
	bctx, err := run.task(rctx, taskArgs)
	if err != nil {
		log.Fatal(err)
	}

	pub, err := getPublisher()
	if err != nil {
		log.Fatal(err)
	}

	if pub != nil {
		if err := pub.publish(rctx, bctx, target.publisherParams); err != nil {
			log.Fatal(err)
		}
	}
}
