package runners

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"

	"github.com/rafaelmartins/yatr/internal/executils"
	"github.com/rafaelmartins/yatr/internal/runners/autotools"
	"github.com/rafaelmartins/yatr/internal/runners/dwtk"
	"github.com/rafaelmartins/yatr/internal/runners/golang"
	"github.com/rafaelmartins/yatr/internal/runners/script"
	"github.com/rafaelmartins/yatr/internal/types"
)

type Runner interface {
	Name() string
	Detect(ctx *types.Ctx) bool
	Configure(ctx *types.Ctx, args []string) (*types.Project, error)
	Task(ctx *types.Ctx, proj *types.Project, args []string) error
	Collect(ctx *types.Ctx, proj *types.Project, args []string) ([]string, error)
}

var runners = []Runner{
	&autotools.AutotoolsRunner{},
	&golang.GolangRunner{},
	&dwtk.DwtkRunner{},
	&script.ScriptRunner{},
}

func Get(targetName string, srcDir string, buildDir string) (Runner, *types.Ctx) {
	ctx := &types.Ctx{
		TargetName: targetName,
		SrcDir:     srcDir,
		BuildDir:   buildDir,
	}

	// ensure build dir is clean
	os.RemoveAll(ctx.BuildDir)
	os.MkdirAll(ctx.BuildDir, 0777)

	for _, v := range runners {
		if v.Detect(ctx) {
			return v, ctx
		}
	}

	return nil, nil
}

func RunTargetScript(ctx *types.Ctx, proj *types.Project, taskScript string, taskArgs []string) error {
	if !path.IsAbs(taskScript) {
		taskScript = filepath.Join(ctx.SrcDir, taskScript)
	}
	cmd := exec.Command(taskScript, taskArgs...)
	cmd.Dir = ctx.BuildDir
	cmd.Env = append(
		os.Environ(),
		fmt.Sprintf("SRCDIR=%s", ctx.SrcDir),
		fmt.Sprintf("BUILDDIR=%s", ctx.BuildDir),
		fmt.Sprintf("PN=%s", proj.Name),
		fmt.Sprintf("PV=%s", proj.Version),
		fmt.Sprintf("P=%s-%s", proj.Name, proj.Version),
		fmt.Sprintf("MAKE_CMD=make -j%d", runtime.NumCPU()+1),
	)
	return executils.Run(cmd)
}
