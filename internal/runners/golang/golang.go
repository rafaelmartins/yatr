package golang

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/rafaelmartins/yatr/internal/compress"
	"github.com/rafaelmartins/yatr/internal/executils"
	"github.com/rafaelmartins/yatr/internal/fs"
	"github.com/rafaelmartins/yatr/internal/git"
	"github.com/rafaelmartins/yatr/internal/types"
)

var validOSArch = []string{
	"android-arm",
	"darwin-386",
	"darwin-amd64",
	"darwin-armv5",
	"darwin-armv6",
	"darwin-armv7",
	"darwin-arm64",
	"dragonfly-amd64",
	"freebsd-386",
	"freebsd-amd64",
	"freebsd-armv5",
	"freebsd-armv6",
	"freebsd-armv7",
	"linux-386",
	"linux-amd64",
	"linux-armv5",
	"linux-armv6",
	"linux-armv7",
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
	"netbsd-armv5",
	"netbsd-armv6",
	"netbsd-armv7",
	"openbsd-386",
	"openbsd-amd64",
	"openbsd-armv5",
	"openbsd-armv6",
	"openbsd-armv7",
	"plan9-386",
	"plan9-amd64",
	"solaris-amd64",
	"windows-386",
	"windows-amd64",
}

var validDistTarget = regexp.MustCompile(`^dist-(([a-z0-9]+)-([a-z0-9]+))$`)
var mainPackage = regexp.MustCompile(`^[ \t]*package[ \t]+main[ \t]*$`)

type GolangRunner struct {
	GoTool    string
	IsWindows bool
	OsArch    string
	Binaries  []string
}

func supportModules() bool {
	cmd := exec.Command("go", "help", "mod")
	cmd.Stdout = ioutil.Discard
	cmd.Stderr = ioutil.Discard
	return cmd.Run() == nil
}

func generateFullLicense(ctx *types.Ctx) (bool, error) {
	gomodFile := filepath.Join(ctx.SrcDir, "go.mod")
	vendorDir := filepath.Join(ctx.SrcDir, "vendor")

	// create vendor directory if not available
	if _, err := os.Stat(gomodFile); err == nil {
		if _, err := os.Stat(vendorDir); os.IsNotExist(err) {
			cmd := exec.Command("go", "mod", "vendor")
			cmd.Dir = ctx.SrcDir
			if err := executils.Run(cmd); err != nil {
				return false, err
			}
		}
	}

	// get main license
	mainLicense := fs.FindLicense(ctx.SrcDir)
	if len(mainLicense) == 0 {
		return false, nil
	}

	f, err := os.Create(filepath.Join(ctx.BuildDir, "license.txt"))
	if err != nil {
		return false, err
	}
	defer f.Close()

	content, err := ioutil.ReadFile(filepath.Join(ctx.SrcDir, mainLicense))
	if err != nil {
		return false, err
	}

	if _, err := f.Write(content); err != nil {
		return false, err
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

	return true, nil
}

func getMainPackages(ctx *types.Ctx) []string {
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

func (r *GolangRunner) Name() string {
	return "golang"
}

func (r *GolangRunner) Detect(ctx *types.Ctx) bool {
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

	return found
}

func (r *GolangRunner) Configure(ctx *types.Ctx, args []string) (*types.Project, error) {
	// guess project name
	projectName := path.Base(ctx.SrcDir)

	// guess project version
	projectVersion := git.Version(ctx.SrcDir)

	if m := supportModules(); m {
		os.Setenv("GO111MODULE", "on")
	}

	return &types.Project{Name: projectName, Version: projectVersion}, nil
}

func (r *GolangRunner) Task(ctx *types.Ctx, proj *types.Project, args []string) error {
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

		goArch := matches[3]
		goArm := ""
		if strings.HasPrefix(matches[3], "armv") {
			goArch = "arm"
			goArm = matches[3][4:]
		}

		for _, dir := range getMainPackages(ctx) {
			goArgs := append([]string{r.GoTool, "-v", "-x"}, args...)
			goArgs = append(goArgs, dir)
			cmd := exec.Command("go", goArgs...)
			cmd.Dir = ctx.BuildDir
			cmd.Env = append(
				os.Environ(),
				fmt.Sprintf("GOOS=%s", matches[2]),
				fmt.Sprintf("GOARCH=%s", goArch),
			)
			if goArm != "" {
				cmd.Env = append(
					cmd.Env,
					fmt.Sprintf("GOARM=%s", goArm),
				)
			}
			if err := executils.Run(cmd); err != nil {
				return err
			}

			r.Binaries = append(r.Binaries, path.Base(dir))
		}
	} else {
		cmd := exec.Command("go", append([]string{r.GoTool, "-v"}, args...)...)
		cmd.Dir = ctx.SrcDir
		return executils.Run(cmd)
	}

	return nil
}

func (r *GolangRunner) Collect(ctx *types.Ctx, proj *types.Project, args []string) ([]string, error) {
	var builtFiles []string

	if r.GoTool == "build" {

		toCompress := []string{}

		for _, binaryName := range r.Binaries {
			if r.IsWindows {
				binaryName = fmt.Sprintf("%s.exe", proj.Name)
			}
			toCompress = append(toCompress, binaryName)
		}

		license, err := generateFullLicense(ctx)
		if err != nil {
			return nil, err
		}
		if license {
			toCompress = append(toCompress, "license.txt")
		}

		readme := fs.FindReadme(ctx.SrcDir)
		if len(readme) > 0 {
			readmeSrc := filepath.Join(ctx.SrcDir, readme)
			readmeDst := filepath.Join(ctx.BuildDir, "readme.txt")
			if err := fs.CopyFile(readmeSrc, readmeDst); err != nil {
				return nil, err
			}
			toCompress = append(toCompress, "readme.txt")
		}

		fileExtension := "tar.gz"
		if r.IsWindows {
			fileExtension = "zip"
		}
		filePrefix := fmt.Sprintf("%s-%s-%s", proj.Name, r.OsArch, proj.Version)
		fileName := fmt.Sprintf("%s.%s", filePrefix, fileExtension)

		filePath := filepath.Join(ctx.BuildDir, fileName)
		f, err := os.Create(filePath)
		if err != nil {
			return nil, err
		}
		defer f.Close()

		if r.IsWindows {
			if err := compress.Zip(ctx.BuildDir, filePrefix, toCompress, f); err != nil {
				return nil, err
			}
		} else {
			if err := compress.TarGzip(ctx.BuildDir, filePrefix, toCompress, f); err != nil {
				return nil, err
			}
		}

		builtFiles = []string{fileName}
	}

	return builtFiles, nil
}
