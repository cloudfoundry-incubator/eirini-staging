package eirinistaging

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

type DropletUploader struct {
	Client *http.Client
}

func (u *DropletUploader) Upload(
	dropletUploadURL string,
	dropletLocation string,
) error {
	if dropletLocation == "" {
		return errors.New("empty path parameter")
	}
	if dropletUploadURL == "" {
		return errors.New("empty url parameter")
	}

	return u.uploadFile(dropletLocation, dropletUploadURL)
}

func (u *DropletUploader) uploadFile(fileLocation, url string) error {
	sourceFile, err := os.Open(filepath.Clean(fileLocation))
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}

	body := ioutil.NopCloser(sourceFile)
	request, err := http.NewRequest("POST", url, body)
	if err != nil {
		return fmt.Errorf("failed to create http request: %w", err)
	}

	contentLength, err := fileSize(sourceFile)
	if err != nil {
		return err
	}

	request.ContentLength = contentLength
	request.Header.Set("Content-Type", "application/octet-stream")

	return u.do(request)
}

func fileSize(file *os.File) (int64, error) {
	fileInfo, err := file.Stat()
	if err != nil {
		return 0, fmt.Errorf("failed to stat file: %w", err)
	}

	return fileInfo.Size(), nil
}

func (u *DropletUploader) do(req *http.Request) error {
	resp, err := u.Client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("Upload failed: Status code %d", resp.StatusCode)
	}

	return nil
}
