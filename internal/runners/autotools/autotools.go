package autotools

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/rafaelmartins/yatr/internal/executils"
	"github.com/rafaelmartins/yatr/internal/types"
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

type AutotoolsRunner struct{}

func getAutotoolsProject(ctx *types.Ctx) *types.Project {
	projectName := "UNKNOWN"
	projectVersion := "UNKNOWN"

	content, err := ioutil.ReadFile(filepath.Join(ctx.BuildDir, "config.log"))
	if err != nil {
		return &types.Project{Name: projectName, Version: projectVersion}
	}

	matches := configLogNameVersion.FindAllStringSubmatch(string(content), -1)
	if matches == nil {
		return &types.Project{Name: projectName, Version: projectVersion}
	}

	for _, match := range matches {
		if match[1] == "TARNAME" {
			projectName = match[2]
		} else if match[1] == "VERSION" {
			projectVersion = match[2]
		}
	}

	return &types.Project{Name: projectName, Version: projectVersion}
}

func (r *AutotoolsRunner) Name() string {
	return "autotools"
}

func (r *AutotoolsRunner) Detect(ctx *types.Ctx) bool {
	path := path.Join(ctx.SrcDir, "configure.ac")
	_, err := os.Stat(path)
	return err == nil
}

func (r *AutotoolsRunner) Configure(ctx *types.Ctx, args []string) (*types.Project, error) {
	cmd := exec.Command("autoreconf", "--warnings=all", "--install", "--force")
	cmd.Dir = ctx.SrcDir
	if err := executils.Run(cmd); err != nil {
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

	cmd = exec.Command(configure, args...)
	cmd.Dir = ctx.BuildDir
	err = executils.Run(cmd)

	return getAutotoolsProject(ctx), err
}

func (r *AutotoolsRunner) Task(ctx *types.Ctx, proj *types.Project, args []string) error {
	jobs := fmt.Sprintf("-j%d", runtime.NumCPU()+1)
	makeArgs := append(append([]string{jobs}, args...), ctx.TargetName)

	cmd := exec.Command("make", makeArgs...)
	cmd.Dir = ctx.BuildDir
	return executils.Run(cmd)
}

func (r *AutotoolsRunner) Collect(ctx *types.Ctx, proj *types.Project, args []string) ([]string, error) {
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
