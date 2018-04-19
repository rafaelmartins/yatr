package main

import (
	"bytes"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
)

type distfilesApiPublisher struct {
	url string
}

func (p *distfilesApiPublisher) name() string {
	return "distfiles-api"
}

func (p *distfilesApiPublisher) publish(rctx runnerCtx, bctx buildCtx, pattern string) error {
	log.Println("Step: Publish (Publisher: distfiles-api)")

	for _, archive := range bctx.archives {
		log.Println("    - Uploading archive:", archive)

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

		extract := false
		if len(pattern) > 0 {
			var err error
			extract, err = filepath.Match(pattern, archive)
			if err != nil {
				extract = false
			}
		}

		extractStr := "false"
		if extract {
			extractStr = "true"
		}

		reqParams := map[string]string{
			"project": bctx.projectName,
			"version": bctx.projectVersion,
			"sha512":  checksumStr,
			"extract": extractStr,
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
				"Error: Failed to upload file to distfiles api: %s (%d: %s)",
				archive,
				resp.StatusCode,
				bodyString,
			)
		}

		log.Println("          Done!")
	}
	return nil
}
