package main

import (
	"bufio"
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
var mainPackage = regexp.MustCompile(`^[ \t]*package[ \t]+main[ \t]*$`)

type golangRunner struct {
	goTool    string
	isWindows bool
	osArch    string
	Binaries  []string
}

func getFullLicense(ctx runnerCtx) string {
	gomodFile := filepath.Join(ctx.srcDir, "go.mod")
	vendorDir := filepath.Join(ctx.srcDir, "vendor")

	// create vendor directory if not available
	if _, err := os.Stat(gomodFile); err == nil {
		if _, err := os.Stat(vendorDir); os.IsNotExist(err) {
			cmd := command(ctx.srcDir, "go", "mod", "vendor")
			cmd.Env = append(
				os.Environ(),
				"GO111MODULE=on",
			)
			run(cmd)
		}
	}

	// get main license
	mainLicense := getLicense(ctx.srcDir)
	if len(mainLicense) == 0 {
		return ""
	}

	f, err := os.Create(filepath.Join(ctx.buildDir, mainLicense))
	if err != nil {
		return ""
	}
	defer f.Close()

	content, err := ioutil.ReadFile(filepath.Join(ctx.srcDir, mainLicense))
	if err != nil {
		return ""
	}

	if _, err := f.Write(content); err != nil {
		return ""
	}

	filepath.Walk(vendorDir, func(path string, info os.FileInfo, err error) error {
		if err == nil && info.IsDir() {
			license := getLicense(path)
			if len(license) > 0 {
				repo := strings.TrimPrefix(path, vendorDir+string(os.PathSeparator))

				f.WriteString("\n\n\n#### License for ")
				f.WriteString(repo)
				f.WriteString(":\n\n")

				content, err := ioutil.ReadFile(filepath.Join(path, license))
				if err != nil {
					return err
				}

				if _, err := f.Write(content); err != nil {
					return err
				}
			}
		}
		return nil
	})

	return mainLicense
}

func getMainPackages(ctx runnerCtx) []string {
	rv := []string{}

	filepath.Walk(ctx.srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && info.Name() == "vendor" {
			return filepath.SkipDir
		}
		if info.Mode().IsRegular() && strings.HasSuffix(path, ".go") {
			f, err := os.Open(path)
			if err != nil {
				return err
			}
			defer f.Close()

			scanner := bufio.NewScanner(f)
			for scanner.Scan() {
				if mainPackage.MatchString(scanner.Text()) {
					c := filepath.Dir(path)

					found := false
					for _, dir := range rv {
						if dir == c {
							found = true
						}
					}
					if !found {
						rv = append(rv, c)
					}
				}
			}

			if err := scanner.Err(); err != nil {
				return err
			}
		}

		return nil
	})

	return rv
}

func (r *golangRunner) name() string {
	return "golang"
}

func (r *golangRunner) configure(ctx runnerCtx, args []string) (project, error) {
	log.Println("Step: Configure (Runner: golang)")

	// guess project name
	projectName := path.Base(ctx.srcDir)

	// guess project version
	projectVersion := gitVersion(ctx.srcDir)

	return project{Name: projectName, Version: projectVersion}, nil
}

func (r *golangRunner) task(ctx runnerCtx, proj project, args []string) error {
	log.Println("Step: Task (Runner: golang)")

	if ctx.targetName == "distcheck" {
		r.goTool = "test"
	} else if strings.HasPrefix(ctx.targetName, "dist-") {
		r.goTool = "build"
	} else {
		return fmt.Errorf("Error: Target not supported for golang: %s", ctx.targetName)
	}

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

		for _, dir := range getMainPackages(ctx) {
			goArgs := append([]string{r.goTool, "-v", "-x", dir}, args...)
			cmd := command(ctx.srcDir, "go", goArgs...)
			cmd.Env = append(
				os.Environ(),
				fmt.Sprintf("GOOS=%s", matches[2]),
				fmt.Sprintf("GOARCH=%s", matches[3]),
			)
			if err := run(cmd); err != nil {
				return err
			}

			r.Binaries = append(r.Binaries, path.Base(dir))
		}
	} else {
		goArgs := append([]string{r.goTool, "-v", "-x"}, args...)
		cmd := command(ctx.srcDir, "go", goArgs...)
		return run(cmd)
	}

	return nil
}

func (r *golangRunner) collect(ctx runnerCtx, proj project, args []string) ([]string, error) {
	log.Println("Step: Collect (Runner: golang)")

	var builtFiles []string

	if r.goTool == "build" {

		toCompress := []string{}

		for _, binaryName := range r.Binaries {
			if r.isWindows {
				binaryName = fmt.Sprintf("%s.exe", proj.Name)
			}
			binaryPath := path.Join(ctx.srcDir, binaryName)

			if st, err := os.Stat(binaryPath); err == nil && st.Mode()&0111 != 0 {
				if err := os.Rename(binaryPath, path.Join(ctx.buildDir, binaryName)); err != nil {
					return nil, err
				}
			}

			toCompress = append(toCompress, binaryName)
		}

		license := getFullLicense(ctx)
		if len(license) > 0 {
			toCompress = append(toCompress, license)
		}

		readme := getReadme(ctx.srcDir)
		if len(readme) > 0 {
			copyFile(
				filepath.Join(ctx.srcDir, readme),
				filepath.Join(ctx.buildDir, readme),
			)
			toCompress = append(toCompress, readme)
		}

		fileExtension := "tar.gz"
		if r.isWindows {
			fileExtension = "zip"
		}
		filePrefix := fmt.Sprintf("%s-%s-%s", proj.Name, r.osArch, proj.Version)
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

	return builtFiles, nil
}
