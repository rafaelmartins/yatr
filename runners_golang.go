package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
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

type golangRunner struct {
	goTool    string
	isWindows bool
	osArch    string
}

func (r *golangRunner) name() string {
	return "golang"
}

func (r *golangRunner) configure(ctx *runnerCtx, args []string) error {
	return nil
}

func (r *golangRunner) task(ctx *runnerCtx, args []string) error {
	log.Println("Step: Task (Runner: golang)")

	if ctx.targetName == "distcheck" {
		r.goTool = "test"
	} else if strings.HasPrefix(ctx.targetName, "dist-") {
		r.goTool = "build"
	} else {
		return fmt.Errorf("Error: Target not supported for golang: %s", ctx.targetName)
	}

	goArgs := append([]string{r.goTool, "-v", "-x"}, args...)
	cmd := command(ctx.srcDir, "go", goArgs...)

	r.isWindows = false

	if r.goTool == "build" {
		matches := validDistTarget.FindStringSubmatch(ctx.targetName)
		if matches == nil {
			return fmt.Errorf("Error: Invalid target name for golang: %s", ctx.targetName)
		}

		r.osArch = matches[1]

		found := false
		for _, elem := range validOSArch {
			if r.osArch == elem {
				found = true
			}
		}
		if !found {
			return fmt.Errorf("Error: Unsupported dist target for golang: %s", ctx.targetName)
		}

		r.isWindows = matches[2] == "windows"

		cmd.Env = append(
			os.Environ(),
			fmt.Sprintf("GOOS=%s", matches[2]),
			fmt.Sprintf("GOARCH=%s", matches[3]),
		)
	}

	return cmd.Run()
}

func (r *golangRunner) collect(ctx *runnerCtx, args []string) (*buildCtx, error) {
	log.Println("Step: Collect (Runner: golang)")

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

	if r.goTool == "build" {

		binaryName := buildName
		if r.isWindows {
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
		if r.isWindows {
			fileExtension = "zip"
		}
		filePrefix := fmt.Sprintf("%s-%s-%s", buildName, r.osArch, buildVersion)
		fileName := fmt.Sprintf("%s.%s", filePrefix, fileExtension)

		var data []byte
		if r.isWindows {
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
	}

	return &buildCtx{projectName: buildName, projectVersion: buildVersion, archives: builtFiles}, nil
}
