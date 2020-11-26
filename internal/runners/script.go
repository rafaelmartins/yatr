package runners

import (
	"fmt"
	"io/ioutil"
	"path"
	"time"

	"github.com/rafaelmartins/yatr/internal/git"
)

type scriptRunner struct{}

func (s *scriptRunner) Name() string {
	return "script"
}

func (s *scriptRunner) Detect(ctx *Ctx) bool {
	return true
}

func (s *scriptRunner) Configure(ctx *Ctx, args []string) (*Project, error) {
	projectName := path.Base(ctx.SrcDir)

	t := "version-git"
	for _, arg := range args {
		if arg == "version-date" || arg == "version-unix" || arg == "version-git" {
			t = arg
		}
	}

	projectVersion := ""
	switch t {
	case "version-date":
		n := time.Now().UTC()
		h, m, _ := n.Clock()
		y, mo, d := n.Date()
		projectVersion = fmt.Sprintf("%04d%02d%02d%02d%02d", y, mo, d, h, m)
	case "version-unix":
		projectVersion = fmt.Sprintf("%d", time.Now().Unix())
	case "version-git":
		projectVersion = git.Version(ctx.SrcDir)
	}

	return &Project{Name: projectName, Version: projectVersion}, nil
}

func (s *scriptRunner) Task(ctx *Ctx, proj *Project, args []string) error {
	return fmt.Errorf("script runner does not have a default task, please set task_script in your config")
}

func (s *scriptRunner) Collect(ctx *Ctx, proj *Project, args []string) ([]string, error) {
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
