package distfiles_api

import (
	"bytes"
	"crypto/sha512"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/rafaelmartins/yatr/internal/runners"
)

type DistfilesApiPublisher struct {
	Url string
}

func (p *DistfilesApiPublisher) Name() string {
	return "distfiles-api"
}

func (p *DistfilesApiPublisher) Detect(ctx *runners.Ctx) bool {
	if url := strings.TrimSpace(os.Getenv("DISTFILES_URL")); url != "" {
		p.Url = url
		return true
	}
	return false
}

func (p *DistfilesApiPublisher) Publish(ctx *runners.Ctx, proj *runners.Project, archives []string, pattern string) error {
	for _, archive := range archives {
		log.Println("    - Uploading archive:", archive)

		fn := filepath.Join(ctx.BuildDir, archive)

		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)
		part, err := writer.CreateFormFile("file", archive)
		if err != nil {
			return err
		}

		checksum := sha512.New()

		if err := func() error {
			f, err := os.Open(fn)
			if err != nil {
				return err
			}
			defer f.Close()

			_, err = io.Copy(io.MultiWriter(part, checksum), f)
			return err
		}(); err != nil {
			return err
		}

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
			"sha512":  fmt.Sprintf("%x  %s", checksum.Sum(nil), archive),
			"extract": extractStr,
		}

		for key, value := range reqParams {
			if err := writer.WriteField(key, value); err != nil {
				return err
			}
		}

		if err := writer.Close(); err != nil {
			return err
		}

		resp, err := http.Post(p.Url, writer.FormDataContentType(), body)
		if err != nil {
			return err
		}

		bodyContent, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

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
