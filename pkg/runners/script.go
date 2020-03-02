package runners

import (
	"fmt"
	"io/ioutil"
	"log"
	"path"
	"time"
)

type scriptRunner struct{}

func (s *scriptRunner) Name() string {
	return "script"
}

func (s *scriptRunner) Detect(ctx *Ctx) Runner {
	return s
}

func (s *scriptRunner) Configure(ctx *Ctx, args []string) (*Project, error) {
	log.Println("Step: Configure (Runner: script)")
	projectName := path.Base(ctx.SrcDir)
	projectVersion := fmt.Sprintf("%d", time.Now().Unix())
	return &Project{Name: projectName, Version: projectVersion}, nil
}

func (s *scriptRunner) Task(ctx *Ctx, proj *Project, args []string) error {
	return fmt.Errorf("script runner does not have a default task, please set task_script in your config")
}

func (s *scriptRunner) Collect(ctx *Ctx, proj *Project, args []string) ([]string, error) {
	log.Println("Step: Collect (Runner: script)")

	files, err := ioutil.ReadDir(ctx.BuildDir)
	if err != nil {
		return nil, err
	}

	var rv []string
	for _, fileInfo := range files {
		if !fileInfo.Mode().IsRegular() {
			continue
		}
		rv = append(rv, fileInfo.Name())
	}
	return rv, nil
}