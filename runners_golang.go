package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
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
	if ctx.TargetName == "distcheck" {
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
	var osArch string

	if goTool == "build" {
		matches := validDistTarget.FindStringSubmatch(ctx.TargetName)
		if matches == nil {
			return nil, fmt.Errorf("error: Invalid target name for golang: %s", ctx.TargetName)
		}

		osArch = matches[1]

		found := false
		for _, elem := range validOSArch {
			if osArch == elem {
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

	// guess build name
	var sources []string
	for _, arg := range args {
		if strings.HasSuffix(arg, ".go") {
			sources = append(sources, arg)
		}
	}
	var buildName string
	if len(sources) == 0 {
		buildName = path.Base(ctx.SrcDir)
	} else {
		buildName = strings.TrimSuffix(sources[0], ".go")
	}

	// guess build version
	buildVersion := GitVersion(ctx.SrcDir)

	var builtFiles []string

	if goTool == "build" {

		binaryName := buildName
		if isWindows {
			binaryName = fmt.Sprintf("%s.exe", buildName)
		}
		binaryPath := path.Join(ctx.SrcDir, binaryName)

		if st, err := os.Stat(binaryPath); err == nil && st.Mode()&0111 != 0 {
			if err := os.Rename(binaryPath, path.Join(ctx.BuildDir, binaryName)); err != nil {
				return nil, err
			}
		}

		toCompress := []string{binaryName}

		license := GetLicense(ctx.SrcDir)
		if license != nil {
			CopyFile(
				filepath.Join(ctx.SrcDir, *license),
				filepath.Join(ctx.BuildDir, *license),
			)
			toCompress = append(toCompress, *license)
		}

		readme := GetReadme(ctx.SrcDir)
		if readme != nil {
			CopyFile(
				filepath.Join(ctx.SrcDir, *readme),
				filepath.Join(ctx.BuildDir, *readme),
			)
			toCompress = append(toCompress, *readme)
		}

		fileExtension := "tar.gz"
		if isWindows {
			fileExtension = "zip"
		}
		filePrefix := fmt.Sprintf("%s-%s-%s", buildName, osArch, buildVersion)
		fileName := fmt.Sprintf("%s.%s", filePrefix, fileExtension)

		var data []byte
		if isWindows {
			var err error
			if data, err = CreateZip(ctx.BuildDir, filePrefix, toCompress); err != nil {
				return nil, err
			}
		} else {
			var err error
			if data, err = CreateTarGz(ctx.BuildDir, filePrefix, toCompress); err != nil {
				return nil, err
			}
		}

		filePath := filepath.Join(ctx.BuildDir, fileName)
		if err := ioutil.WriteFile(filePath, data, 0666); err != nil {
			return nil, err
		}

		builtFiles = []string{fileName}

	} else if goTool == "test" {

		// for test builds we will ship source tarballs
		archiveName := fmt.Sprintf("%s-%s", buildName, buildVersion)
		archives, err := GitArchive(archiveName, ctx.SrcDir, ctx.BuildDir)
		if err != nil {
			return nil, err
		}

		builtFiles = append(builtFiles, archives...)
	}

	return builtFiles, nil
}
