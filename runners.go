package main

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
)

type RunnerCtx struct {
	TargetName string
	SrcDir     string
	BuildDir   string
}

type Runner interface {
	Configure(ctx *RunnerCtx, args []string) error
	Task(ctx *RunnerCtx, args []string) ([]string, error)
}

func GetRunner(targetName string, srcDir string, buildDir string) (Runner, *RunnerCtx, error) {
	ctx := RunnerCtx{
		TargetName: targetName,
		SrcDir:     srcDir,
		BuildDir:   buildDir,
	}

	// ensure build dir is clean
	os.RemoveAll(ctx.BuildDir)
	os.MkdirAll(ctx.BuildDir, 0777)

	// autotools check
	path := path.Join(ctx.SrcDir, "configure.ac")
	if _, err := os.Stat(path); err == nil {
		return &AutotoolsRunner{}, &ctx, nil
	}

	// golang check
	if paths, err := filepath.Glob(fmt.Sprintf("%s/*.go", ctx.SrcDir)); err == nil && len(paths) > 0 {
		return &GolangRunner{}, &ctx, nil
	}

	return nil, nil, fmt.Errorf("error: No runner found!")
}
