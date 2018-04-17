package main

import (
	"log"
	"os"
	"path"
)

func main() {
	log.SetFlags(0)
	log.SetPrefix("[YATR] >>> ")

	log.Println("Starting YATR ...")

	if os.Getenv("TRAVIS") != "true" {
		log.Fatalln("Error: This tool only supports Travis-CI")
	}

	conf, err := configRead(".yatr.yml")
	if err != nil {
		log.Fatal(err)
	}

	targetName, ok := os.LookupEnv("TARGET")
	if !ok {
		log.Fatalln("Error: Target not provided, export TARGET environment variable.")
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

	log.Println("Step: Git repository unshallow")
	gitUnshallow(rctx.srcDir)

	configureArgs := append(conf.DefaultConfigureArgs, target.ConfigureArgs...)
	if err := run.configure(rctx, configureArgs); err != nil {
		log.Fatal(err)
	}

	taskArgs := append(conf.DefaultTaskArgs, target.TaskArgs...)
	taskErr := run.task(rctx, taskArgs)

	bctx, err := run.collect(rctx, taskArgs)
	if err != nil {
		log.Println("Warning:", err)
	}

	if bctx != nil {
		if len(target.ArchiveFilter) > 0 {
			bctx.archives = filterArchives(bctx.archives, target.ArchiveFilter)
		}

		if len(bctx.archives) > 0 {
			log.Println("")
			log.Println("Build details:")
			log.Println("")
			log.Println("    Project Name:   ", bctx.projectName)
			log.Println("    Project Version:", bctx.projectVersion)
			log.Println("    Archives:")
			for _, archive := range bctx.archives {
				log.Println("        -", archive)
			}
			log.Println("")

			if pub != nil {
				if err := pub.publish(rctx, bctx, target.ArchiveExtractFilter); err != nil {
					log.Fatal(err)
				}
			} else {
				log.Println("Step: Publish (Disabled, no publisher available for this build)")
			}
		} else {
			log.Println("Step: Publish (Disabled, no archives to upload)")
		}
	}

	log.Println("")
	if taskErr != nil {
		log.Fatal("!!! TASK FAILED !!!")
	} else {
		log.Println("All done! \\o/")
	}
}
