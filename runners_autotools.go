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
	"tar.gz",
	"tar.bz2",
	"tar.xz",
	"zip",
	"lzip",
}

var configLogNameVersion = regexp.MustCompile(`PACKAGE_(TARNAME|VERSION) *= *['"](.*)['"]`)

type autotoolsRunner struct{}

func configLogGetNameVersion(ctx *runnerCtx) (string, string) {
	name := "UNKNOWN"
	version := "UNKNOWN"

	content, err := ioutil.ReadFile(filepath.Join(ctx.buildDir, "config.log"))
	if err != nil {
		return name, version
	}

	matches := configLogNameVersion.FindAllStringSubmatch(string(content), -1)
	if matches == nil {
		return name, version
	}

	for _, match := range matches {
		if match[1] == "TARNAME" {
			name = match[2]
		} else if match[1] == "VERSION" {
			version = match[2]
		}
	}

	return name, version
}

func (r *autotoolsRunner) name() string {
	return "autotools"
}

func (r *autotoolsRunner) configure(ctx *runnerCtx, args []string) error {
	log.Println("Step: Configure (Runner: autotools)")

	cmd := Command(ctx.srcDir, "autoreconf", "--warnings=all", "--install", "--force")
	if err := cmd.Run(); err != nil {
		return nil
	}

	configure := path.Join(ctx.srcDir, "configure")

	st, err := os.Stat(configure)
	if os.IsNotExist(err) {
		return fmt.Errorf("Error: `configure` script was not created")
	}
	if err != nil {
		return err
	}

	if st.Mode()&0111 == 0 {
		return fmt.Errorf("Error: `configure` script is not executable")
	}

	cmd = Command(ctx.buildDir, configure, args...)
	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

func (r *autotoolsRunner) task(ctx *runnerCtx, args []string) (*buildCtx, error) {
	log.Println("Step: Task (Runner: autotools)")

	jobs := fmt.Sprintf("-j%d", runtime.NumCPU()+1)
	makeArgs := append(append([]string{jobs}, args...), ctx.targetName)

	cmd := Command(ctx.buildDir, "make", makeArgs...)
	if err := cmd.Run(); err != nil {
		return nil, err
	}

	buildName, buildVersion := configLogGetNameVersion(ctx)

	var candidates []string
	files, err := ioutil.ReadDir(ctx.buildDir)
	if err != nil {
		return nil, err
	}
	for _, fileInfo := range files {
		if !fileInfo.Mode().IsRegular() {
			continue
		}
		if !strings.HasPrefix(fileInfo.Name(), fmt.Sprintf("%s-", buildName)) {
			continue
		}
		candidates = append(candidates, fileInfo.Name())
	}

	var builtFiles []string

	for _, ext := range autotoolsDistExts {
		suffix := fmt.Sprintf("%s.%s", buildVersion, ext)
		for _, candidate := range candidates {
			if strings.HasSuffix(candidate, suffix) {
				builtFiles = append(builtFiles, candidate)
			}
		}
	}

	return &buildCtx{projectName: buildName, projectVersion: buildVersion, archives: builtFiles}, nil
}
