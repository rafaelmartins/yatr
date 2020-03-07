package runners

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	execStd "os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/rafaelmartins/yatr/pkg/compress"
	"github.com/rafaelmartins/yatr/pkg/exec"
	"github.com/rafaelmartins/yatr/pkg/fs"
	"github.com/rafaelmartins/yatr/pkg/git"
)

var reTarball = regexp.MustCompile(`"(avr-toolchain-([a-z0-9]+)-([a-z0-9]+)-([0-9]+)\.tar\.xz)"`)

type dwtkRunner struct{}

func (d *dwtkRunner) Name() string {
	return "dwtk"
}

func (d *dwtkRunner) Detect(ctx *Ctx) Runner {
	path := filepath.Join(ctx.SrcDir, "dwtk-config.mk")
	if _, err := os.Stat(path); err == nil {
		return d
	}

	return nil
}

func (d *dwtkRunner) Configure(ctx *Ctx, args []string) (*Project, error) {
	log.Println("Step: Configure (Runner: dwtk)")
	return &Project{
		Name:    filepath.Base(ctx.SrcDir),
		Version: git.Version(ctx.SrcDir),
	}, nil
}

func (d *dwtkRunner) Task(ctx *Ctx, proj *Project, args []string) error {
	log.Println("Step: Task (Runner: dwtk)")

	if ctx.TargetName != "dist-avr" && ctx.TargetName != "dist-avr-debug" {
		return fmt.Errorf("unsupported target: %s", ctx.TargetName)
	}

	path := ""
	if _, err := execStd.LookPath("avr-gcc"); err != nil { // no toolchain found
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

		if err := exec.Run(exec.Cmd(ctx.BuildDir, "wget", url)); err != nil {
			return err
		}
		if err := exec.Run(exec.Cmd(ctx.BuildDir, "tar", "-xvf", file)); err != nil {
			return err
		}

		path = filepath.Join(ctx.BuildDir, "avr", "bin")
		if p, found := os.LookupEnv("PATH"); found {
			path = path + string(filepath.ListSeparator) + p
		}
	}

	root := filepath.Join(ctx.BuildDir, "__root__")

	jobs := fmt.Sprintf("-j%d", runtime.NumCPU()+1)
	makeArgs := append([]string{jobs}, args...)
	cmd := exec.Cmd(ctx.SrcDir, "make", makeArgs...)
	cmd.Env = append(
		os.Environ(),
		fmt.Sprintf("BUILDDIR=%s", root),
	)
	if path != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("PATH=%s", path))
	}

	p := fmt.Sprintf("%s-%s", proj.Name, proj.Version)
	if ctx.TargetName == "dist-avr" {
		cmd.Env = append(cmd.Env, "AVR_RELEASE=1")
	} else {
		p = fmt.Sprintf("%s-debug", p)
	}

	if err := exec.Run(cmd); err != nil {
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

	licenseSrc := fs.FindLicense(ctx.SrcDir)
	if licenseSrc != "" {
		licenseDst := filepath.Join(root, filepath.Base(licenseSrc))
		if err := fs.CopyFile(licenseSrc, licenseDst); err != nil {
			return err
		}
		toCompress = append(toCompress, filepath.Base(licenseSrc))
	}

	filePath := filepath.Join(ctx.BuildDir, fmt.Sprintf("%s.tar.gz", p))
	f, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	return compress.TarGzip(root, p, toCompress, f)
}

func (d *dwtkRunner) Collect(ctx *Ctx, proj *Project, args []string) ([]string, error) {
	p := fmt.Sprintf("%s-%s", proj.Name, proj.Version)
	if ctx.TargetName == "dist-avr-debug" {
		p = fmt.Sprintf("%s-debug", p)
	}

	return []string{fmt.Sprintf("%s.tar.gz", p)}, nil
}
