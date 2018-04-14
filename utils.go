package main

import (
	"io"
	"os"
	"path/filepath"
)

var licenseFiles = []string{
	"LICENSE",
	"LICENCE",
	"UNLICENSE",
	"COPYING",
	"COPYRIGHT",
}

var readmeFiles = []string{
	"README",
	"README.md",
}

func getLicense(dir string) *string {
	for _, entry := range licenseFiles {
		if _, err := os.Stat(filepath.Join(dir, entry)); err == nil {
			return &entry
		}
	}
	return nil
}

func getReadme(dir string) *string {
	for _, entry := range readmeFiles {
		if _, err := os.Stat(filepath.Join(dir, entry)); err == nil {
			return &entry
		}
	}
	return nil
}

func copyFile(srcName string, dstName string) error {
	src, err := os.Open(srcName)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(dstName)
	if err != nil {
		return err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return err
	}

	return nil
}
