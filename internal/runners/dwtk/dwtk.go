package dwtk

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/rafaelmartins/yatr/internal/compress"
	"github.com/rafaelmartins/yatr/internal/executils"
	"github.com/rafaelmartins/yatr/internal/fs"
	"github.com/rafaelmartins/yatr/internal/git"
	"github.com/rafaelmartins/yatr/internal/types"
)

var (
	reTarball   = regexp.MustCompile(`"(avr-toolchain-([a-z0-9]+)-([a-z0-9]+)-([0-9]+)\.tar\.xz)"`)
	reAvrTarget = regexp.MustCompile(`^dist-([a-z0-9]+)(-debug)?$`)
)

type DwtkRunner struct {
	Prefix string
}

func (d *DwtkRunner) Name() string {
	return "dwtk"
}

func (d *DwtkRunner) Detect(ctx *types.Ctx) bool {
	path := filepath.Join(ctx.SrcDir, "dwtk-config.mk")
	_, err := os.Stat(path)
	return err == nil
}

func (d *DwtkRunner) Configure(ctx *types.Ctx, args []string) (*types.Project, error) {
	return &types.Project{
		Name:    filepath.Base(ctx.SrcDir),
		Version: git.Version(ctx.SrcDir),
	}, nil
}

func (d *DwtkRunner) Task(ctx *types.Ctx, proj *types.Project, args []string) error {
	matches := reAvrTarget.FindStringSubmatch(ctx.TargetName)
	if len(matches) == 0 {
		return fmt.Errorf("unsupported target: %s", ctx.TargetName)
	}

	mcu := matches[1]
	release := len(matches[2]) == 0

	path := ""
	if _, err := exec.LookPath("avr-gcc"); err != nil { // no toolchain found
		resp, err := http.Get("https://distfiles.rgm.io/avr-toolchain/LATEST/")
		if err != nil {
			return err
		}
		data, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return err
		}

		url := ""
		file := ""
		matches := reTarball.FindAllStringSubmatch(string(data), -1)
		for _, m := range matches {
			if m[2] == runtime.GOOS || m[3] == runtime.GOARCH {
				url = fmt.Sprintf("https://distfiles.rgm.io/avr-toolchain/avr-toolchain-%s/%s", m[4], m[1])
				file = m[1]
			}
		}
		if url == "" {
			return fmt.Errorf("no toolchain found")
		}

		cmd := exec.Command("wget", url)
		cmd.Dir = ctx.BuildDir
		if err := executils.Run(cmd); err != nil {
			return err
		}

		// fixme: decompress natively
		cmd = exec.Command("tar", "-xvf", file)
		cmd.Dir = ctx.BuildDir
		if err := executils.Run(cmd); err != nil {
			return err
		}

		path = filepath.Join(ctx.BuildDir, "avr", "bin")
		if p, found := os.LookupEnv("PATH"); found {
			path = path + string(filepath.ListSeparator) + p
		}
	}

	root := filepath.Join(ctx.BuildDir, "__root__")

	jobs := fmt.Sprintf("-j%d", runtime.NumCPU()+1)
	cmd := exec.Command("make", append([]string{jobs}, args...)...)
	cmd.Dir = ctx.SrcDir
	cmd.Env = append(
		os.Environ(),
		fmt.Sprintf("BUILDDIR=%s", root),
	)
	if path != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("PATH=%s", path))
	}

	d.Prefix = proj.Name

	if mcu != "avr" {
		d.Prefix += "-" + mcu
		cmd.Env = append(cmd.Env, fmt.Sprintf("AVR_MCU=%s", mcu))
	}

	if release {
		cmd.Env = append(cmd.Env, "AVR_RELEASE=1")
	} else {
		d.Prefix += "-debug"
	}

	d.Prefix += "-" + proj.Version

	if err := executils.Run(cmd); err != nil {
		return err
	}

	toCompress := []string{}

	files, err := ioutil.ReadDir(root)
	if err != nil {
		return err
	}

	for _, fileInfo := range files {
		if !fileInfo.Mode().IsRegular() {
			continue
		}
		if n := fileInfo.Name(); strings.HasSuffix(n, ".hex") || strings.HasSuffix(n, ".elf") {
			toCompress = append(toCompress, n)
		}
	}

	license := fs.FindLicense(ctx.SrcDir)
	if license != "" {
		licenseSrc := filepath.Join(ctx.SrcDir, license)
		licenseDst := filepath.Join(root, "license.txt")
		if err := fs.CopyFile(licenseSrc, licenseDst); err != nil {
			return err
		}
		toCompress = append(toCompress, "license.txt")
	}

	readme := fs.FindReadme(ctx.SrcDir)
	if readme != "" {
		readmeSrc := filepath.Join(ctx.SrcDir, readme)
		readmeDst := filepath.Join(root, "readme.txt")
		if err := fs.CopyFile(readmeSrc, readmeDst); err != nil {
			return err
		}
		toCompress = append(toCompress, "readme.txt")
	}

	filePath := filepath.Join(ctx.BuildDir, fmt.Sprintf("%s.tar.gz", d.Prefix))
	f, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	return compress.TarGzip(root, d.Prefix, toCompress, f)
}

func (d *DwtkRunner) Collect(ctx *types.Ctx, proj *types.Project, args []string) ([]string, error) {
	return []string{fmt.Sprintf("%s.tar.gz", d.Prefix)}, nil
}
