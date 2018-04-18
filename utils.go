package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

var licenseFiles = []string{
	"LICENSE",
	"LICENCE",
	"UNLICENSE",
	"COPYING",
	"COPYRIGHT",
}

var readmeFiles = []string{
	"README",
	"README.md",
}

func getLicense(dir string) *string {
	for _, entry := range licenseFiles {
		if _, err := os.Stat(filepath.Join(dir, entry)); err == nil {
			return &entry
		}
	}
	return nil
}

func getReadme(dir string) *string {
	for _, entry := range readmeFiles {
		if _, err := os.Stat(filepath.Join(dir, entry)); err == nil {
			return &entry
		}
	}
	return nil
}

func copyFile(srcName string, dstName string) error {
	src, err := os.Open(srcName)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(dstName)
	if err != nil {
		return err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return err
	}

	return nil
}

func command(dir string, name string, arg ...string) *exec.Cmd {
	args := append([]string{"    Running command:", name}, arg...)
	log.Println(strings.Join(args, " "))
	log.Println("          Directory:", dir)
	rv := exec.Command(name, arg...)
	rv.Stdout = os.Stdout
	rv.Stderr = os.Stderr
	rv.Dir = dir
	return rv
}

func filterArchives(archives []string, pattern string) []string {
	rv := []string{}
	for _, archive := range archives {
		if matched, err := filepath.Match(pattern, archive); err == nil && matched {
			rv = append(rv, archive)
		}
	}
	return rv
}

func runTargetScript(ctx *runnerCtx, taskScript string, taskArgs []string) error {
	if !path.IsAbs(taskScript) {
		taskScript = filepath.Join(ctx.srcDir, taskScript)
	}
	cmd := command(ctx.buildDir, taskScript, taskArgs...)
	cmd.Env = append(
		os.Environ(),
		fmt.Sprintf("SRCDIR=%s", ctx.srcDir),
		fmt.Sprintf("BUILDDIR=%s", ctx.buildDir),
	)
	return cmd.Run()
}
