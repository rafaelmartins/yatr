package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"strings"
)

func gitVersion(repoDir string) string {
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

func gitArchive(name string, repoDir string, outputDir string) ([]string, error) {
	var archives []string
	prefix := fmt.Sprintf("%s/", name)
	for _, format := range []string{"tar", "zip"} {
		var out bytes.Buffer
		cmd := exec.Command("git", "archive", "--format", format, "--prefix", prefix, "HEAD")
		cmd.Dir = repoDir
		cmd.Stdout = &out
		if err := cmd.Run(); err != nil {
			return nil, err
		}

		archiveContent := out.Bytes()

		ext := format
		if format == "tar" {
			var err error
			archiveContent, err = compressGzip(archiveContent)
			if err != nil {
				return nil, err
			}
			ext = "tar.gz"
		}

		archiveName := fmt.Sprintf("%s.%s", name, ext)
		archivePath := filepath.Join(outputDir, archiveName)
		if err := ioutil.WriteFile(archivePath, archiveContent, 0666); err != nil {
			return nil, err
		}

		archives = append(archives, archiveName)
	}

	return archives, nil
}

func gitUnshallow(repoDir string) error {
	cmd := exec.Command("git", "fetch", "--unshallow")
	cmd.Dir = repoDir
	return cmd.Run()
}
