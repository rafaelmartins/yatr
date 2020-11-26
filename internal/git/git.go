package git

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/rafaelmartins/yatr/internal/executils"
)

func Version(repoDir string) string {
	var out bytes.Buffer
	cmd := exec.Command("git", "describe", "--abbrev=4", "HEAD")
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
	if _, err := os.Stat(filepath.Join(repoDir, ".git", "shallow")); err != nil {
		return nil // not a shallow repo, everything is fine
	}

	cmd := exec.Command("git", "fetch", "--unshallow")
	cmd.Dir = repoDir
	if err := executils.Run(cmd); err != nil {
		return err
	}

	cmd = exec.Command("git", "fetch", "--tags", "--force")
	cmd.Dir = repoDir
	return executils.Run(cmd)
}
