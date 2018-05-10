package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

var autotoolsDistExts = []string{
	"gz",
	"bz2",
	"xz",
	"zip",
	"lzip",
	"rpm",
	"deb",
}

var configLogNameVersion = regexp.MustCompile(`PACKAGE_(TARNAME|VERSION) *= *['"](.*)['"]`)

type autotoolsRunner struct{}

func getAutotoolsProject(ctx runnerCtx) project {
	projectName := "UNKNOWN"
	projectVersion := "UNKNOWN"

	content, err := ioutil.ReadFile(filepath.Join(ctx.buildDir, "config.log"))
	if err != nil {
		return project{Name: projectName, Version: projectVersion}
	}

	matches := configLogNameVersion.FindAllStringSubmatch(string(content), -1)
	if matches == nil {
		return project{Name: projectName, Version: projectVersion}
	}

	for _, match := range matches {
		if match[1] == "TARNAME" {
			projectName = match[2]
		} else if match[1] == "VERSION" {
			projectVersion = match[2]
		}
	}

	return project{Name: projectName, Version: projectVersion}
}

func (r *autotoolsRunner) name() string {
	return "autotools"
}

func (r *autotoolsRunner) configure(ctx runnerCtx, args []string) (project, error) {
	log.Println("Step: Configure (Runner: autotools)")

	cmd := command(ctx.srcDir, "autoreconf", "--warnings=all", "--install", "--force")
	if err := run(cmd); err != nil {
		return project{}, err
	}
	log.Println("")

	configure := path.Join(ctx.srcDir, "configure")

	st, err := os.Stat(configure)
	if os.IsNotExist(err) {
		return project{}, fmt.Errorf("Error: `configure` script was not created")
	}
	if err != nil {
		return project{}, err
	}

	if st.Mode()&0111 == 0 {
		return project{}, fmt.Errorf("Error: `configure` script is not executable")
	}

	cmd = command(ctx.buildDir, configure, args...)

	rv := run(cmd)

	return getAutotoolsProject(ctx), rv
}

func (r *autotoolsRunner) task(ctx runnerCtx, proj project, args []string) error {
	log.Println("Step: Task (Runner: autotools)")

	jobs := fmt.Sprintf("-j%d", runtime.NumCPU()+1)
	makeArgs := append(append([]string{jobs}, args...), ctx.targetName)

	return run(command(ctx.buildDir, "make", makeArgs...))
}

func (r *autotoolsRunner) collect(ctx runnerCtx, proj project, args []string) ([]string, error) {
	log.Println("Step: Collect (Runner: autotools)")

	files, err := ioutil.ReadDir(ctx.buildDir)
	if err != nil {
		return nil, err
	}

	var candidates []string
	for _, fileInfo := range files {
		if !fileInfo.Mode().IsRegular() {
			continue
		}
		if !strings.HasPrefix(fileInfo.Name(), fmt.Sprintf("%s-", proj.Name)) {
			continue
		}
		candidates = append(candidates, fileInfo.Name())
	}

	var builtFiles []string
	for _, ext := range autotoolsDistExts {
		suffix := fmt.Sprintf(".%s", ext)
		for _, candidate := range candidates {
			if strings.HasSuffix(candidate, suffix) {
				builtFiles = append(builtFiles, candidate)
			}
		}
	}

	return builtFiles, nil
}
