package fs

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

func FindLicense(dir string) string {
	for _, entry := range licenseFiles {
		if _, err := os.Stat(filepath.Join(dir, entry)); err == nil {
			return entry
		}
	}
	return ""
}

func FindReadme(dir string) string {
	for _, entry := range readmeFiles {
		if _, err := os.Stat(filepath.Join(dir, entry)); err == nil {
			return entry
		}
	}
	return ""
}

func CopyFile(srcName string, dstName string) error {
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

func FilterArchives(archives []string, pattern string) []string {
	rv := []string{}
	for _, archive := range archives {
		if matched, err := filepath.Match(pattern, archive); err == nil && matched {
			rv = append(rv, archive)
		}
	}
	return rv
}
