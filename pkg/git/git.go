package git

import (
	"bytes"
	"os"
	goExec "os/exec"
	"path/filepath"
	"strings"

	"github.com/rafaelmartins/yatr/pkg/exec"
)

func Version(repoDir string) string {
	var out bytes.Buffer
	cmd := goExec.Command("git", "describe", "--abbrev=4", "HEAD")
	cmd.Dir = repoDir
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "UNKNOWN"
	}

	version := out.String()

	// remove prefix 'v'
	version = strings.TrimPrefix(version, "v")

	// replace first '-' with '.'
	version = strings.Replace(version, "-", ".", 1)

	// remove prefix 'g' from git hash
	version = strings.Replace(version, "-g", "-", 1)

	return strings.Trim(version, " \t\n")
}

func Unshallow(repoDir string) error {
	if _, err := os.Stat(filepath.Join(repoDir, ".git", "shallow")); err == nil {
		rv := exec.Run(exec.Cmd(repoDir, "git", "fetch", "--unshallow"))
		if rv != nil {
			return rv
		}
		return exec.Run(exec.Cmd(repoDir, "git", "fetch", "--tags"))
	}
	return nil
}
