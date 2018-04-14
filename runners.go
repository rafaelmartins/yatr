package main

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
)

type runnerCtx struct {
	targetName string
	srcDir     string
	buildDir   string
}

type buildCtx struct {
	projectName    string
	projectVersion string
	archives       []string
}

type runner interface {
	configure(ctx *runnerCtx, args []string) error
	task(ctx *runnerCtx, args []string) (*buildCtx, error)
}

func getRunner(targetName string, srcDir string, buildDir string) (runner, *runnerCtx, error) {
	ctx := runnerCtx{
		targetName: targetName,
		srcDir:     srcDir,
		buildDir:   buildDir,
	}

	// ensure build dir is clean
	os.RemoveAll(ctx.buildDir)
	os.MkdirAll(ctx.buildDir, 0777)

	// autotools check
	path := path.Join(ctx.srcDir, "configure.ac")
	if _, err := os.Stat(path); err == nil {
		return &autotoolsRunner{}, &ctx, nil
	}

	// golang check
	if paths, err := filepath.Glob(fmt.Sprintf("%s/*.go", ctx.srcDir)); err == nil && len(paths) > 0 {
		return &golangRunner{}, &ctx, nil
	}

	return nil, nil, fmt.Errorf("error: No runner found!")
}
