package main

import (
	"fmt"
	"log"
	"os"
	"path"
)

func main() {
	log.SetFlags(0)
	log.SetPrefix("[YATR] >>> ")

	banner := "Starting YATR"
	conf, err := configRead(".yatr.yml")
	if err == nil {
		banner = fmt.Sprintf("%s (Using configuration file: .yatr.yml)", banner)
	}
	log.Println(banner, "...")

	targetName, ok := os.LookupEnv("TARGET")
	if !ok {
		log.Fatalln("Error: Target not provided, export TARGET environemnt variable.")
	}

	if os.Getenv("TRAVIS") != "true" {
		log.Fatalln("Error: This tool only supports Travis-CI")
	}

	log.Println("")
	log.Println("    Target:   ", targetName)

	target := conf.Targets[targetName]

	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	run, rctx := getRunner(targetName, dir, path.Join(dir, "build"))
	if run == nil {
		log.Fatal("Error: No runner found for this project!")
	}
	log.Println("    Runner:   ", run.name())

	pub := getPublisher()
	if pub == nil {
		log.Println("    Publisher: (not available)")
	} else {
		log.Println("    Publisher:", pub.name())
	}

	log.Println("")
	log.Println("    Source directory:", rctx.srcDir)
	log.Println("    Build directory: ", rctx.buildDir)
	log.Println("")

	configureArgs := append(conf.DefaultConfigureArgs, target.ConfigureArgs...)
	if err := run.configure(rctx, configureArgs); err != nil {
		log.Fatal(err)
	}

	taskArgs := append(conf.DefaultTaskArgs, target.TaskArgs...)
	bctx, err := run.task(rctx, taskArgs)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("")
	log.Println("Build details:")
	log.Println("")
	log.Println("    Project Name:", bctx.projectName)
	log.Println("    Project Version:", bctx.projectVersion)
	if len(bctx.archives) > 0 {
		log.Println("    Archives:")
		for _, archive := range bctx.archives {
			log.Println("        - ", archive)
		}
	}
	log.Println("")

	if pub != nil {
		if len(bctx.archives) > 0 {
			if err := pub.publish(rctx, bctx, target.PublisherParams); err != nil {
				log.Fatal(err)
			}
		} else {
			log.Println("Step: Publish (Disabled, no archives to upload)")
		}
	} else {
		log.Println("Step: Publish (Disabled, no publisher available for this build)")
	}

	log.Println("")
	log.Println("All done! \\o/")
}
