package runners

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

	"github.com/rafaelmartins/yatr/internal/exec"
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

func getAutotoolsProject(ctx *Ctx) *Project {
	projectName := "UNKNOWN"
	projectVersion := "UNKNOWN"

	content, err := ioutil.ReadFile(filepath.Join(ctx.BuildDir, "config.log"))
	if err != nil {
		return &Project{Name: projectName, Version: projectVersion}
	}

	matches := configLogNameVersion.FindAllStringSubmatch(string(content), -1)
	if matches == nil {
		return &Project{Name: projectName, Version: projectVersion}
	}

	for _, match := range matches {
		if match[1] == "TARNAME" {
			projectName = match[2]
		} else if match[1] == "VERSION" {
			projectVersion = match[2]
		}
	}

	return &Project{Name: projectName, Version: projectVersion}
}

func (r *autotoolsRunner) Name() string {
	return "autotools"
}

func (r *autotoolsRunner) Detect(ctx *Ctx) bool {
	path := path.Join(ctx.SrcDir, "configure.ac")
	_, err := os.Stat(path)
	return err == nil
}

func (r *autotoolsRunner) Configure(ctx *Ctx, args []string) (*Project, error) {
	cmd := exec.Cmd(ctx.SrcDir, "autoreconf", "--warnings=all", "--install", "--force")
	if err := exec.Run(cmd); err != nil {
		return nil, err
	}
	log.Println("")

	configure := path.Join(ctx.SrcDir, "configure")

	st, err := os.Stat(configure)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("Error: `configure` script was not created")
	}
	if err != nil {
		return nil, err
	}

	if st.Mode()&0111 == 0 {
		return nil, fmt.Errorf("Error: `configure` script is not executable")
	}

	cmd = exec.Cmd(ctx.BuildDir, configure, args...)

	rv := exec.Run(cmd)

	return getAutotoolsProject(ctx), rv
}

func (r *autotoolsRunner) Task(ctx *Ctx, proj *Project, args []string) error {
	jobs := fmt.Sprintf("-j%d", runtime.NumCPU()+1)
	makeArgs := append(append([]string{jobs}, args...), ctx.TargetName)

	return exec.Run(exec.Cmd(ctx.BuildDir, "make", makeArgs...))
}

func (r *autotoolsRunner) Collect(ctx *Ctx, proj *Project, args []string) ([]string, error) {
	files, err := ioutil.ReadDir(ctx.BuildDir)
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
