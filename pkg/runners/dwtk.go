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
				url = "https://distfiles.rgm.io/avr-toolchain/avr-toolchain-" + m[4] + "/" + m[1]
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

	jobs := fmt.Sprintf("-j%d", runtime.NumCPU()+1)
	makeArgs := append([]string{jobs}, args...)
	cmd := exec.Cmd(ctx.SrcDir, "make", makeArgs...)
	cmd.Env = os.Environ()

	p := proj.Name + "-" + proj.Version
	if ctx.TargetName == "dist-avr" {
		cmd.Env = append(cmd.Env, "AVR_RELEASE=1")
	} else {
		p += "-debug"
	}
	cmd.Env = append(cmd.Env, fmt.Sprintf("BUILDDIR=%s", filepath.Join(ctx.BuildDir, p)))

	if path != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("PATH=%s", path))
	}

	if err := exec.Run(cmd); err != nil {
		return err
	}

	licenseSrc := fs.FindLicense(ctx.SrcDir)
	if licenseSrc != "" {
		license := filepath.Join(ctx.BuildDir, p, filepath.Base(licenseSrc))
		if err := fs.CopyFile(licenseSrc, license); err != nil {
			return err
		}
	}

	return exec.Run(exec.Cmd(ctx.BuildDir, "tar", "-cvzf", p+".tar.gz", p))
}

func (d *dwtkRunner) Collect(ctx *Ctx, proj *Project, args []string) ([]string, error) {
	p := proj.Name + "-" + proj.Version
	if ctx.TargetName == "dist-avr-debug" {
		p += "-debug"
	}

	return []string{p + ".tar.gz"}, nil
}
