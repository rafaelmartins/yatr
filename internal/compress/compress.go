package compress

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func copyToWriter(filename string, w io.Writer) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(w, f)
	return err
}

func TarGzip(chdir string, prefix string, entries []string, out io.Writer) error {
	gz := gzip.NewWriter(out)
	defer gz.Close()
	tw := tar.NewWriter(gz)
	defer tw.Close()

	for _, entry := range entries {
		entryFile := filepath.Join(chdir, entry)

		info, err := os.Stat(entryFile)
		if err != nil {
			return err
		}

		if !info.Mode().IsRegular() {
			continue
		}

		hdrName := fmt.Sprintf("%s/%s", prefix, entry)
		hdr, err := tar.FileInfoHeader(info, hdrName)
		if err != nil {
			return err
		}

		hdr.Name = hdrName

		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}

		if err := copyToWriter(entryFile, tw); err != nil {
			return err
		}
	}

	return nil
}

func Zip(chdir string, prefix string, entries []string, out io.Writer) error {
	zw := zip.NewWriter(out)
	defer zw.Close()

	for _, entry := range entries {
		entryFile := filepath.Join(chdir, entry)

		info, err := os.Stat(entryFile)
		if err != nil {
			return err
		}

		if !info.Mode().IsRegular() {
			continue
		}

		f, err := zw.Create(fmt.Sprintf("%s/%s", prefix, entry))
		if err != nil {
			return err
		}

		if err := copyToWriter(entryFile, f); err != nil {
			return err
		}
	}

	return nil
}
