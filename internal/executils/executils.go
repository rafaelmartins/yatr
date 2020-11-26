package executils

import (
	"log"
	"os"
	"os/exec"
	"syscall"
)

func Run(cmd *exec.Cmd) error {
	log.Printf("    Running command: %q", append([]string{cmd.Path}, cmd.Args...))
	if cmd.Dir != "" {
		log.Println("          Directory:", cmd.Dir)
	}

	if cmd.Stdout == nil {
		cmd.Stdout = os.Stdout
	}
	if cmd.Stderr == nil {
		cmd.Stdout = os.Stderr
	}

	err := cmd.Run()

	code := 0
	if err != nil {
		if msg, ok := err.(*exec.ExitError); ok {
			if ws, ok := msg.Sys().(syscall.WaitStatus); ok {
				code = ws.ExitStatus()
			}
		}
	}
	log.Println("          Exit code:", code)
	return err
}
