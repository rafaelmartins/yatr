package publishers

import (
	"bytes"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/rafaelmartins/yatr/pkg/runners"
)

type distfilesApiPublisher struct {
	Url string
}

func (p *distfilesApiPublisher) Name() string {
	return "distfiles-api"
}

func (p *distfilesApiPublisher) Detect(ctx *runners.Ctx) bool {
	distfilesApiUrl, found := os.LookupEnv("DISTFILES_URL")
	if found && len(distfilesApiUrl) > 0 {
		p.Url = distfilesApiUrl
	}
	return found
}

func (p *distfilesApiPublisher) Publish(ctx *runners.Ctx, proj *runners.Project, archives []string, pattern string) error {
	for _, archive := range archives {
		log.Println("    - Uploading archive:", archive)

		f := filepath.Join(ctx.BuildDir, archive)

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
			"project": proj.Name,
			"version": proj.Version,
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

		resp, err := http.Post(p.Url, contentType, body)
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
