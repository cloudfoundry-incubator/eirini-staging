package eirinistaging

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

type PackageInstaller struct {
	client      *http.Client
	downloadURL string
	downloadDir string
}

func NewPackageManager(client *http.Client, downloadURL, downloadDir string) Installer {
	return &PackageInstaller{
		client:      client,
		downloadURL: downloadURL,
		downloadDir: downloadDir,
	}
}

func (d *PackageInstaller) Install() error {
	if d.downloadURL == "" {
		return errors.New("empty downloadURL provided")
	}

	if d.downloadDir == "" {
		return errors.New("empty downloadDir provided")
	}

	downloadPath := filepath.Join(d.downloadDir, AppBits)
	err := d.download(d.downloadURL, downloadPath)
	if err != nil {
		return err
	}

	return nil
}

func (d *PackageInstaller) download(downloadURL string, filepath string) error {
	file, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	resp, err := d.client.Get(downloadURL)
	if err != nil {
		return errors.Wrap(err, "failed to perform request")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.New(fmt.Sprintf("download failed. status code %d", resp.StatusCode))
	}

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return errors.Wrap(err, "failed to copy content to file")
	}

	return nil
}
