package exec

import (
	"log"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

func Cmd(dir string, name string, arg ...string) *exec.Cmd {
	args := append([]string{"    Running command:", name}, arg...)
	log.Println(strings.Join(args, " "))
	log.Println("          Directory:", dir)
	rv := exec.Command(name, arg...)
	rv.Stdout = os.Stdout
	rv.Stderr = os.Stderr
	rv.Dir = dir
	return rv
}

func Run(cmd *exec.Cmd) error {
	code := 0
	err := cmd.Run()
	if err != nil {
		if msg, ok := err.(*exec.ExitError); ok {
			code = msg.Sys().(syscall.WaitStatus).ExitStatus()
		}
	}
	log.Println("          Exit code:", code)
	return err
}
