package main

import (
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
	name() string
	configure(ctx *runnerCtx, args []string) error
	task(ctx *runnerCtx, args []string) error
	collect(ctx *runnerCtx, args []string) (*buildCtx, error)
}

func getRunner(targetName string, srcDir string, buildDir string) (runner, *runnerCtx) {
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
		return &autotoolsRunner{}, &ctx
	}

	// golang check
	if paths, err := filepath.Glob(filepath.Join(ctx.srcDir, "*.go")); err == nil && len(paths) > 0 {
		return &golangRunner{}, &ctx
	}

	return nil, nil
}
