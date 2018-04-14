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

type golangRunner struct{}

func (r *golangRunner) configure(ctx *runnerCtx, args []string) error {
	return nil
}

func (r *golangRunner) task(ctx *runnerCtx, args []string) ([]string, error) {
	var goTool string
	if ctx.targetName == "distcheck" {
		goTool = "test"
	} else if strings.HasPrefix(ctx.targetName, "dist-") {
		goTool = "build"
	} else {
		return nil, fmt.Errorf("error: Target not supported for golang: %s", ctx.targetName)
	}

	goArgs := append([]string{goTool, "-v", "-x"}, args...)

	cmd := exec.Command("go", goArgs...)
	cmd.Dir = ctx.srcDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	isWindows := false
	var osArch string

	if goTool == "build" {
		matches := validDistTarget.FindStringSubmatch(ctx.targetName)
		if matches == nil {
			return nil, fmt.Errorf("error: Invalid target name for golang: %s", ctx.targetName)
		}

		osArch = matches[1]

		found := false
		for _, elem := range validOSArch {
			if osArch == elem {
				found = true
			}
		}
		if !found {
			return nil, fmt.Errorf("error: Unsupported dist target for golang: %s", ctx.targetName)
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
		buildName = path.Base(ctx.srcDir)
	} else {
		buildName = strings.TrimSuffix(sources[0], ".go")
	}

	// guess build version
	buildVersion := gitVersion(ctx.srcDir)

	var builtFiles []string

	if goTool == "build" {

		binaryName := buildName
		if isWindows {
			binaryName = fmt.Sprintf("%s.exe", buildName)
		}
		binaryPath := path.Join(ctx.srcDir, binaryName)

		if st, err := os.Stat(binaryPath); err == nil && st.Mode()&0111 != 0 {
			if err := os.Rename(binaryPath, path.Join(ctx.buildDir, binaryName)); err != nil {
				return nil, err
			}
		}

		toCompress := []string{binaryName}

		license := getLicense(ctx.srcDir)
		if license != nil {
			copyFile(
				filepath.Join(ctx.srcDir, *license),
				filepath.Join(ctx.buildDir, *license),
			)
			toCompress = append(toCompress, *license)
		}

		readme := getReadme(ctx.srcDir)
		if readme != nil {
			copyFile(
				filepath.Join(ctx.srcDir, *readme),
				filepath.Join(ctx.buildDir, *readme),
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
			if data, err = createZip(ctx.buildDir, filePrefix, toCompress); err != nil {
				return nil, err
			}
		} else {
			var err error
			if data, err = createTarGz(ctx.buildDir, filePrefix, toCompress); err != nil {
				return nil, err
			}
		}

		filePath := filepath.Join(ctx.buildDir, fileName)
		if err := ioutil.WriteFile(filePath, data, 0666); err != nil {
			return nil, err
		}

		builtFiles = []string{fileName}

	} else if goTool == "test" {

		// for test builds we will ship source tarballs
		archiveName := fmt.Sprintf("%s-%s", buildName, buildVersion)
		archives, err := gitArchive(archiveName, ctx.srcDir, ctx.buildDir)
		if err != nil {
			return nil, err
		}

		builtFiles = append(builtFiles, archives...)
	}

	return builtFiles, nil
}
