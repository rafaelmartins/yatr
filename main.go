package main

import (
	"bytes"
	"log"
	"os"
	"path"
	"text/template"

	"github.com/rafaelmartins/yatr/internal/config"
	"github.com/rafaelmartins/yatr/internal/fs"
	"github.com/rafaelmartins/yatr/internal/git"
	"github.com/rafaelmartins/yatr/internal/publishers"
	"github.com/rafaelmartins/yatr/internal/runners"
)

func main() {
	log.SetFlags(0)
	log.SetPrefix("[YATR] >>> ")

	log.Println("Starting YATR ...")
	log.Println("")

	conf, err := config.Read(".yatr.yml")
	if err != nil {
		log.Fatal("Error: ", err)
	}

	targetName, ok := os.LookupEnv("TARGET")
	if !ok {
		log.Fatalln("Error: Target not provided, export TARGET environment variable.")
	}

	log.Println("    Target:   ", targetName)

	target := conf.Targets[targetName]

	dir, err := os.Getwd()
	if err != nil {
		log.Fatal("Error: ", err)
	}

	run, ctx := runners.Get(targetName, dir, path.Join(dir, "build"))
	if run == nil || ctx == nil {
		log.Fatal("Error: No runner found for this project!")
	}
	log.Println("    Runner:   ", run.Name())

	pub, pubErr := publishers.Get(ctx)
	if pubErr != nil {
		log.Printf("    Publisher: (%s)", pubErr)
	} else if pub != nil {
		log.Println("    Publisher:", pub.Name())
	} else {
		log.Println("    Publisher: (not available)")
	}

	log.Println("")
	log.Println("    Source directory:", ctx.SrcDir)
	log.Println("    Build directory: ", ctx.BuildDir)
	log.Println("")

	log.Println("Step: Git repository unshallow")
	if err := git.Unshallow(ctx.SrcDir); err != nil {
		log.Fatal("Error: ", err)
	}
	log.Println("")

	configureArgs := append(conf.DefaultConfigureArgs, target.ConfigureArgs...)

	log.Printf("Step: Configure (Runner: %s)\n", run.Name())
	proj, err := run.Configure(ctx, configureArgs)
	if err != nil {
		log.Fatal("Error: ", err)
	}
	log.Println("")

	tmpl := template.New("task-args")

	taskArgs := append(conf.DefaultTaskArgs, target.TaskArgs...)
	finalTaskArgs := []string{}
	for _, arg := range taskArgs {
		t, err := tmpl.Parse(arg)
		if err != nil {
			log.Fatal("Error: ", err)
		}

		b := new(bytes.Buffer)
		if err := t.Execute(b, proj); err != nil {
			log.Fatal("Error: ", err)
		}

		finalTaskArgs = append(finalTaskArgs, b.String())
	}

	log.Printf("Step: Task (Runner: %s)\n", run.Name())
	var taskErr error
	if len(target.TaskScript) > 0 {
		taskErr = runners.RunTargetScript(ctx, proj, target.TaskScript, finalTaskArgs)
	} else {
		taskErr = run.Task(ctx, proj, finalTaskArgs)
	}
	log.Println("")
	if taskErr != nil && !target.PublishOnFailure {
		log.Fatal("Error: ", taskErr)
	}

	log.Printf("Step: Collect (Runner: %s)\n", run.Name())
	archives, err := run.Collect(ctx, proj, finalTaskArgs)
	if err != nil {
		log.Println("Warning: ", err)
	}
	log.Println("")

	archives = fs.CheckArchives(ctx.BuildDir, archives)

	if len(target.ArchiveFilter) > 0 {
		archives = fs.FilterArchives(archives, target.ArchiveFilter)
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
			log.Printf("Step: Publish (Publisher: %s)\n", pub.Name())
			if err := pub.Publish(ctx, proj, archives, target.ArchiveExtractFilter); err != nil {
				log.Fatal("Error: ", err)
			}
		}
	} else {
		log.Println("Step: Publish (disabled, no archives to upload)")
	}

	log.Println("")
	if taskErr != nil {
		log.Println("!!! TASK FAILED !!!")
		log.Println()
		log.Fatal("Error: ", taskErr)
	} else {
		log.Println("All done! \\o/")
	}
}
