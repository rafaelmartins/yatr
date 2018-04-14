package main

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"runtime"
)

type autotoolsRunner struct{}

func (r *autotoolsRunner) configure(ctx *runnerCtx, args []string) error {
	cmd := exec.Command("autoreconf", "--warnings=all", "--install", "--force")
	cmd.Dir = ctx.srcDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return nil
	}

	configure := path.Join(ctx.srcDir, "configure")

	st, err := os.Stat(configure)
	if os.IsNotExist(err) {
		return fmt.Errorf("error: `configure` script was not created")
	}
	if err != nil {
		return err
	}

	if st.Mode()&0111 == 0 {
		return fmt.Errorf("error: `configure` script is not executable")
	}

	cmd = exec.Command(configure, args...)
	cmd.Dir = ctx.buildDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

func (r *autotoolsRunner) task(ctx *runnerCtx, args []string) ([]string, error) {
	jobs := fmt.Sprintf("-j%d", runtime.NumCPU()+1)
	makeArgs := append([]string{jobs}, args...)
	cmd := exec.Command("make", makeArgs...)
	cmd.Dir = ctx.buildDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return nil, err
	}

	// FIXME: return list of built distfiles
	return nil, nil
}
