package main

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"
)

var validOSArch = []string{
	"android-arm",
	"darwin-386",
	"darwin-amd64",
	"darwin-arm",
	"darwin-arm64",
	"dragonfly-amd64",
	"freebsd-386",
	"freebsd-amd64",
	"freebsd-arm",
	"linux-386",
	"linux-amd64",
	"linux-arm",
	"linux-arm64",
	"linux-ppc64",
	"linux-ppc64le",
	"linux-mips",
	"linux-mipsle",
	"linux-mips64",
	"linux-mips64le",
	"linux-s390x",
	"netbsd-386",
	"netbsd-amd64",
	"netbsd-arm",
	"openbsd-386",
	"openbsd-amd64",
	"openbsd-arm",
	"plan9-386",
	"plan9-amd64",
	"solaris-amd64",
	"windows-386",
	"windows-amd64",
}

var validDistTarget = regexp.MustCompile(`^dist-(([a-z0-9]+)-([a-z0-9]+))$`)

type GolangRunner struct{}

func (r *GolangRunner) Configure(ctx *RunnerCtx, args []string) error {
	return nil
}

func (r *GolangRunner) Task(ctx *RunnerCtx, args []string) ([]string, error) {
	var goTool string
	if ctx.TargetName == "test" {
		goTool = "test"
	} else if strings.HasPrefix(ctx.TargetName, "dist-") {
		goTool = "build"
	} else {
		return nil, fmt.Errorf("error: Target not supported for golang: %s", ctx.TargetName)
	}

	goArgs := append([]string{goTool, "-v", "-x"}, args...)

	cmd := exec.Command("go", goArgs...)
	cmd.Dir = ctx.SrcDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	isWindows := false
	if goTool == "build" {
		matches := validDistTarget.FindStringSubmatch(ctx.TargetName)
		if matches == nil {
			return nil, fmt.Errorf("error: Invalid target name for golang: %s", ctx.TargetName)
		}

		found := false
		for _, elem := range validOSArch {
			if matches[1] == elem {
				found = true
			}
		}
		if !found {
			return nil, fmt.Errorf("error: Unsupported dist target for golang: %s", ctx.TargetName)
		}

		isWindows = matches[2] == "windows"

		cmd.Env = append(
			os.Environ(),
			fmt.Sprintf("GOOS=%s", matches[2]),
			fmt.Sprintf("GOARCH=%s", matches[3]),
		)
	}

	if err := cmd.Run(); err != nil {
		return nil, err
	}

	var builtFiles []string
	if goTool == "build" {

		// guess binary name
		var sources []string
		for _, arg := range args {
			if strings.HasSuffix(arg, ".go") {
				sources = append(sources, arg)
			}
		}
		var bin string
		if len(sources) == 0 {
			bin = path.Base(ctx.SrcDir)
		} else {
			bin = strings.TrimSuffix(sources[0], ".go")
		}
		if isWindows {
			bin = fmt.Sprintf("%s.exe", bin)
		}
		binPath := path.Join(ctx.SrcDir, bin)

		if st, err := os.Stat(binPath); err == nil && st.Mode()&0111 != 0 {
			if err := os.Rename(binPath, path.Join(ctx.BuildDir, bin)); err != nil {
				return nil, err
			}
			builtFiles = []string{bin}
		}
	}

	return builtFiles, nil
}
