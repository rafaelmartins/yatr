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

	conf, err := ConfigRead(".yatr.yml")
	if err != nil {
		log.Fatal(err)
	}
	target := conf.Targets[targetName]

	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	runner, ctx, err := GetRunner(targetName, dir, path.Join(dir, "build"))
	if err != nil {
		log.Fatal(err)
	}

	configureArgs := append(conf.DefaultConfigureArgs, target.ConfigureArgs...)
	if err := runner.Configure(ctx, configureArgs); err != nil {
		log.Fatal(err)
	}

	taskArgs := append(conf.DefaultTaskArgs, target.TaskArgs...)
	builtFiles, err := runner.Task(ctx, taskArgs)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(builtFiles)
}
