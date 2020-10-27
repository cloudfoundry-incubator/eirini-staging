package eirinistaging

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type Unzipper struct {
	UnzippedSizeLimit int64
}

func (u *Unzipper) Extract(src, targetDir string) error {
	if targetDir == "" {
		return errors.New("target directory cannot be empty")
	}

	reader, err := zip.OpenReader(src)
	if err != nil {
		return fmt.Errorf("failed to open zip reader: %w", err)
	}
	defer reader.Close()

	for _, file := range reader.File {
		destPath := filepath.Join(filepath.Clean(targetDir), filepath.Clean(file.Name))

		if file.FileInfo().IsDir() {
			if err = os.MkdirAll(destPath, file.Mode()); err != nil {
				return fmt.Errorf("failed to create dir: %w", err)
			}

			continue
		}

		if err = u.extractFile(file, destPath); err != nil {
			return err
		}
	}

	return err
}

func (u *Unzipper) extractFile(src *zip.File, destPath string) error {
	parentDir := filepath.Dir(destPath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return fmt.Errorf("failed to create dir: %w", err)
	}

	reader, err := src.Open()
	if err != nil {
		return fmt.Errorf("failed to open zip: %w", err)
	}
	defer reader.Close()

	destFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer destFile.Close()

	_, err = io.CopyN(destFile, reader, u.UnzippedSizeLimit)
	if err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("failed to write zip: %w", err)
	}
	if err == nil {
		return fmt.Errorf("extracting zip stopped at %d limit", u.UnzippedSizeLimit)
	}

	return destFile.Chmod(src.Mode())
}
