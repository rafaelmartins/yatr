package main

import (
	"bytes"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
)

type distfilesApiPublisher struct {
	url string
}

func (p *distfilesApiPublisher) publish(rctx *runnerCtx, bctx *buildCtx, params map[string]string) error {
	for _, archive := range bctx.archives {
		f := filepath.Join(rctx.buildDir, archive)

		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)
		part, err := writer.CreateFormFile("file", archive)
		if err != nil {
			return err
		}

		contents, err := ioutil.ReadFile(f)
		if err != nil {
			return err
		}

		part.Write(contents)

		checksum := sha512.Sum512(contents)
		checksumHex := hex.EncodeToString(checksum[:])
		checksumStr := fmt.Sprintf("%s  %s", checksumHex, archive)

		extract, found := params["extract"]
		if !found {
			extract = "false"
		}

		reqParams := map[string]string{
			"project": bctx.projectName,
			"version": bctx.projectVersion,
			"sha512":  checksumStr,
			"extract": extract,
		}

		for key, value := range reqParams {
			if err := writer.WriteField(key, value); err != nil {
				return err
			}
		}

		contentType := writer.FormDataContentType()

		if err := writer.Close(); err != nil {
			return err
		}

		resp, err := http.Post(p.url, contentType, body)
		if err != nil {
			return err
		}

		bodyContent, err := ioutil.ReadAll(resp.Body)
		defer resp.Body.Close()
		if err != nil {
			return err
		}

		bodyString := strings.Trim(string(bodyContent), " \t\n")

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf(
				"error: failed to upload file to distfiles api: %s (%d: %s)",
				archive,
				resp.StatusCode,
				bodyString,
			)
		}
	}
	return nil
}
