package runners

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	goExec "os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/rafaelmartins/yatr/pkg/compress"
	"github.com/rafaelmartins/yatr/pkg/exec"
	"github.com/rafaelmartins/yatr/pkg/fs"
	"github.com/rafaelmartins/yatr/pkg/git"
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
	GoTool    string
	IsWindows bool
	OsArch    string
	Binaries  []string
}

func supportModules() bool {
	cmd := goExec.Command("go", "help", "mod")
	cmd.Stdout = ioutil.Discard
	cmd.Stderr = ioutil.Discard
	return cmd.Run() == nil
}

func getFullLicense(ctx *Ctx) string {
	gomodFile := filepath.Join(ctx.SrcDir, "go.mod")
	vendorDir := filepath.Join(ctx.SrcDir, "vendor")

	// create vendor directory if not available
	if _, err := os.Stat(gomodFile); err == nil {
		if _, err := os.Stat(vendorDir); os.IsNotExist(err) {
			if mod := os.Getenv("GO111MODULE"); mod == "on" {
				exec.Run(exec.Cmd(ctx.SrcDir, "go", "mod", "vendor"))
			}
		}
	}

	// get main license
	mainLicense := fs.FindLicense(ctx.SrcDir)
	if len(mainLicense) == 0 {
		return ""
	}

	f, err := os.Create(filepath.Join(ctx.BuildDir, mainLicense))
	if err != nil {
		return ""
	}
	defer f.Close()

	content, err := ioutil.ReadFile(filepath.Join(ctx.SrcDir, mainLicense))
	if err != nil {
		return ""
	}

	if _, err := f.Write(content); err != nil {
		return ""
	}

	filepath.Walk(vendorDir, func(path string, info os.FileInfo, err error) error {
		if err == nil && info.IsDir() {
			license := fs.FindLicense(path)
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

func getMainPackages(ctx *Ctx) []string {
	rv := []string{}

	filepath.Walk(ctx.SrcDir, func(path string, info os.FileInfo, err error) error {
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

func (r *golangRunner) Name() string {
	return "golang"
}

func (r *golangRunner) Detect(ctx *Ctx) Runner {
	found := false

	filepath.Walk(ctx.SrcDir, func(path string, info os.FileInfo, err error) error {
		if found {
			return nil
		}

		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if filepath.Ext(info.Name()) == ".go" {
			found = true
		}

		return nil
	})

	if found {
		return &golangRunner{}
	}

	return nil
}

func (r *golangRunner) Configure(ctx *Ctx, args []string) (*Project, error) {
	log.Println("Step: Configure (Runner: golang)")

	// guess project name
	projectName := path.Base(ctx.SrcDir)

	// guess project version
	projectVersion := git.Version(ctx.SrcDir)

	if m := supportModules(); m {
		os.Setenv("GO111MODULE", "on")
	}

	return &Project{Name: projectName, Version: projectVersion}, nil
}

func (r *golangRunner) Task(ctx *Ctx, proj *Project, args []string) error {
	log.Println("Step: Task (Runner: golang)")

	if ctx.TargetName == "distcheck" {
		r.GoTool = "test"
	} else if strings.HasPrefix(ctx.TargetName, "dist-") {
		r.GoTool = "build"
	} else {
		return fmt.Errorf("Error: Target not supported for golang: %s", ctx.TargetName)
	}

	r.IsWindows = false

	if r.GoTool == "build" {
		matches := validDistTarget.FindStringSubmatch(ctx.TargetName)
		if matches == nil {
			return fmt.Errorf("Error: Invalid target name for golang: %s", ctx.TargetName)
		}

		r.OsArch = matches[1]

		found := false
		for _, elem := range validOSArch {
			if r.OsArch == elem {
				found = true
			}
		}
		if !found {
			return fmt.Errorf("Error: Unsupported dist target for golang: %s", ctx.TargetName)
		}

		r.IsWindows = matches[2] == "windows"

		for _, dir := range getMainPackages(ctx) {
			goArgs := append([]string{r.GoTool, "-v", "-x"}, args...)
			goArgs = append(goArgs, dir)
			cmd := exec.Cmd(ctx.BuildDir, "go", goArgs...)
			cmd.Env = append(
				os.Environ(),
				fmt.Sprintf("GOOS=%s", matches[2]),
				fmt.Sprintf("GOARCH=%s", matches[3]),
			)
			if err := exec.Run(cmd); err != nil {
				return err
			}

			r.Binaries = append(r.Binaries, path.Base(dir))
		}
	} else {
		goArgs := append([]string{r.GoTool, "-v"}, args...)
		cmd := exec.Cmd(ctx.SrcDir, "go", goArgs...)
		return exec.Run(cmd)
	}

	return nil
}

func (r *golangRunner) Collect(ctx *Ctx, proj *Project, args []string) ([]string, error) {
	log.Println("Step: Collect (Runner: golang)")

	var builtFiles []string

	if r.GoTool == "build" {

		toCompress := []string{}

		for _, binaryName := range r.Binaries {
			if r.IsWindows {
				binaryName = fmt.Sprintf("%s.exe", proj.Name)
			}
			toCompress = append(toCompress, binaryName)
		}

		license := getFullLicense(ctx)
		if len(license) > 0 {
			toCompress = append(toCompress, license)
		}

		readme := fs.FindReadme(ctx.SrcDir)
		if len(readme) > 0 {
			fs.CopyFile(
				filepath.Join(ctx.SrcDir, readme),
				filepath.Join(ctx.BuildDir, readme),
			)
			toCompress = append(toCompress, readme)
		}

		fileExtension := "tar.gz"
		if r.IsWindows {
			fileExtension = "zip"
		}
		filePrefix := fmt.Sprintf("%s-%s-%s", proj.Name, r.OsArch, proj.Version)
		fileName := fmt.Sprintf("%s.%s", filePrefix, fileExtension)

		var data []byte
		if r.IsWindows {
			var err error
			if data, err = compress.Zip(ctx.BuildDir, filePrefix, toCompress); err != nil {
				return nil, err
			}
		} else {
			var err error
			if data, err = compress.TarGzip(ctx.BuildDir, filePrefix, toCompress); err != nil {
				return nil, err
			}
		}

		filePath := filepath.Join(ctx.BuildDir, fileName)
		if err := ioutil.WriteFile(filePath, data, 0666); err != nil {
			return nil, err
		}

		builtFiles = []string{fileName}
	}

	return builtFiles, nil
}