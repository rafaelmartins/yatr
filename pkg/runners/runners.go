package runners

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"

	"github.com/rafaelmartins/yatr/pkg/exec"
)

type Ctx struct {
	TargetName string
	SrcDir     string
	BuildDir   string
}

type Project struct {
	Name    string
	Version string
}

type Runner interface {
	Name() string
	Detect(ctx *Ctx) Runner
	Configure(ctx *Ctx, args []string) (*Project, error)
	Task(ctx *Ctx, proj *Project, args []string) error
	Collect(ctx *Ctx, proj *Project, args []string) ([]string, error)
}

var runners = []Runner{
	&autotoolsRunner{},
	&golangRunner{},
	&dwtkRunner{},
	&scriptRunner{},
}

func Get(targetName string, srcDir string, buildDir string) (Runner, *Ctx) {
	ctx := &Ctx{
		TargetName: targetName,
		SrcDir:     srcDir,
		BuildDir:   buildDir,
	}

	// ensure build dir is clean
	os.RemoveAll(ctx.BuildDir)
	os.MkdirAll(ctx.BuildDir, 0777)

	for _, v := range runners {
		if r := v.Detect(ctx); r != nil {
			return r, ctx
		}
	}

	return nil, nil
}

func RunTargetScript(ctx *Ctx, proj *Project, taskScript string, taskArgs []string) error {
	if !path.IsAbs(taskScript) {
		taskScript = filepath.Join(ctx.SrcDir, taskScript)
	}
	cmd := exec.Cmd(ctx.BuildDir, taskScript, taskArgs...)
	cmd.Env = append(
		os.Environ(),
		fmt.Sprintf("SRCDIR=%s", ctx.SrcDir),
		fmt.Sprintf("BUILDDIR=%s", ctx.BuildDir),
		fmt.Sprintf("PN=%s", proj.Name),
		fmt.Sprintf("PV=%s", proj.Version),
		fmt.Sprintf("P=%s-%s", proj.Name, proj.Version),
		fmt.Sprintf("MAKE_CMD=make -j%d", runtime.NumCPU()+1),
	)
	return exec.Run(cmd)
}
