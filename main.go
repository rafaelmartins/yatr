package main

import (
	"bytes"
	"log"
	"os"
	"path"
	"text/template"
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

	pub, pubErr := getPublisher()
	if pubErr != nil {
		log.Printf("    Publisher: (%s)", pubErr)
	} else if pub != nil {
		log.Println("    Publisher:", pub.name())
	} else {
		log.Println("    Publisher: (not available)")
	}

	log.Println("")
	log.Println("    Source directory:", rctx.srcDir)
	log.Println("    Build directory: ", rctx.buildDir)
	log.Println("")

	log.Println("Step: Git repository unshallow")
	if err := gitUnshallow(rctx.srcDir); err != nil {
		log.Fatal(err)
	}
	log.Println("")

	configureArgs := append(conf.DefaultConfigureArgs, target.ConfigureArgs...)
	proj, err := run.configure(rctx, configureArgs)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("")

	tmpl := template.New("task-args")

	taskArgs := append(conf.DefaultTaskArgs, target.TaskArgs...)
	finalTaskArgs := []string{}
	for _, arg := range taskArgs {
		t, err := tmpl.Parse(arg)
		if err != nil {
			log.Fatal(err)
		}

		b := new(bytes.Buffer)
		if err := t.Execute(b, proj); err != nil {
			log.Fatal(err)
		}

		finalTaskArgs = append(finalTaskArgs, b.String())
	}

	var taskErr error
	if len(target.TaskScript) > 0 {
		taskErr = runTargetScript(rctx, proj, target.TaskScript, finalTaskArgs)
	} else {
		taskErr = run.task(rctx, proj, finalTaskArgs)
	}
	log.Println("")

	archives, err := run.collect(rctx, proj, finalTaskArgs)
	if err != nil {
		log.Println("Warning:", err)
	}
	log.Println("")

	if len(target.ArchiveFilter) > 0 {
		archives = filterArchives(archives, target.ArchiveFilter)
	}

	if len(archives) > 0 {
		log.Println("Build details:")
		log.Println("")
		log.Println("    Project Name:   ", proj.Name)
		log.Println("    Project Version:", proj.Version)
		log.Println("    Archives:")
		for _, archive := range archives {
			log.Println("        -", archive)
		}
		log.Println("")

		if pubErr != nil {
			log.Printf("Step: Publish: (%s)", pubErr)
		} else {
			if err := pub.publish(rctx, proj, archives, target.ArchiveExtractFilter); err != nil {
				log.Fatal(err)
			}
		}
	} else {
		log.Println("Step: Publish (disabled, no archives to upload)")
	}

	log.Println("")
	if taskErr != nil {
		log.Fatal("!!! TASK FAILED !!!")
	} else {
		log.Println("All done! \\o/")
	}
}
