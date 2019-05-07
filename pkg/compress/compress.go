package compress

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

func Gzip(input []byte) ([]byte, error) {
	buf := new(bytes.Buffer)
	gw := gzip.NewWriter(buf)

	if _, err := gw.Write(input); err != nil {
		return nil, err
	}

	if err := gw.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func Tar(chdir string, prefix string, entries []string) ([]byte, error) {
	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)

	for _, entry := range entries {
		entryFile := filepath.Join(chdir, entry)

		info, err := os.Stat(entryFile)
		if err != nil {
			return nil, err
		}

		if !info.Mode().IsRegular() {
			continue
		}

		hdrName := fmt.Sprintf("%s/%s", prefix, entry)
		hdr, err := tar.FileInfoHeader(info, hdrName)
		if err != nil {
			return nil, err
		}

		hdr.Name = hdrName

		if err := tw.WriteHeader(hdr); err != nil {
			return nil, err
		}

		content, err := ioutil.ReadFile(entryFile)
		if err != nil {
			return nil, err
		}

		if _, err := tw.Write(content); err != nil {
			return nil, err
		}

		if err := tw.Flush(); err != nil {
			return nil, err
		}
	}

	if err := tw.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func Zip(chdir string, prefix string, entries []string) ([]byte, error) {
	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)

	for _, entry := range entries {
		entryFile := filepath.Join(chdir, entry)

		info, err := os.Stat(entryFile)
		if err != nil {
			return nil, err
		}

		if !info.Mode().IsRegular() {
			continue
		}

		f, err := zw.Create(fmt.Sprintf("%s/%s", prefix, entry))
		if err != nil {
			return nil, err
		}

		content, err := ioutil.ReadFile(entryFile)
		if err != nil {
			return nil, err
		}

		if _, err := f.Write(content); err != nil {
			return nil, err
		}

		if err := zw.Flush(); err != nil {
			return nil, err
		}
	}

	if err := zw.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func TarGzip(chdir string, prefix string, entries []string) ([]byte, error) {
	t, err := Tar(chdir, prefix, entries)
	if err != nil {
		return nil, err
	}

	x, err := Gzip(t)
	if err != nil {
		return nil, err
	}

	return x, nil
}
