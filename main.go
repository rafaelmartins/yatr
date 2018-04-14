package main

import (
	"fmt"
	"log"
	"os"
	"path"
)

func main() {
	targetName, ok := os.LookupEnv("TARGET")
	if !ok {
		log.Fatalln("error: Target not provided")
	}

	conf, _ := configRead(".yatr.yml")

	target := conf.targets[targetName]

	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	runner, rctx, err := getRunner(targetName, dir, path.Join(dir, "build"))
	if err != nil {
		log.Fatal(err)
	}

	configureArgs := append(conf.defaultConfigureArgs, target.configureArgs...)
	if err := runner.configure(rctx, configureArgs); err != nil {
		log.Fatal(err)
	}

	taskArgs := append(conf.defaultTaskArgs, target.taskArgs...)
	bctx, err := runner.task(rctx, taskArgs)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(bctx)
}
