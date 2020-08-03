package eirinistaging

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"code.cloudfoundry.org/eirini-staging/builder"
	"github.com/pkg/errors"
)

type BuildpackManager struct {
	unzipper       Unzipper
	buildpackDir   string
	buildpacksJSON string
	internalClient *http.Client
	defaultClient  *http.Client
}

const configFileName = "config.json"

func OpenBuildpackURL(buildpackURL string, client *http.Client) ([]byte, error) {
	resp, err := client.Get(buildpackURL)
	if err != nil {
		return nil, errors.Wrap(err, "failed to request buildpack")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(fmt.Sprintf("downloading buildpack failed with status code %d", resp.StatusCode))
	}

	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return bytes, nil
}

func NewBuildpackManager(internalClient *http.Client, defaultClient *http.Client, buildpackDir, buildpacksJSON string) Installer {
	var tenGB int64 = 10 * 1024 * 1024 * 1024

	return &BuildpackManager{
		internalClient: internalClient,
		defaultClient:  defaultClient,
		buildpackDir:   buildpackDir,
		buildpacksJSON: buildpacksJSON,
		unzipper:       Unzipper{UnzippedSizeLimit: tenGB},
	}
}

func (b *BuildpackManager) Install() error {
	var buildpacks []builder.Buildpack

	err := json.Unmarshal([]byte(b.buildpacksJSON), &buildpacks)
	if err != nil {
		fmt.Printf("Error unmarshaling environment variable %s: %s\n", b.buildpacksJSON, err.Error())

		return err
	}

	for _, buildpack := range buildpacks {
		if err := b.install(buildpack); err != nil {
			return fmt.Errorf("installing buildpack %s: %s failed: %s", buildpack.Name, buildpack.URL, err.Error())
		}
	}

	return b.writeBuildpackJSON(buildpacks)
}

func (b *BuildpackManager) install(buildpack builder.Buildpack) error {
	destination := builder.BuildpackPath(b.buildpackDir, buildpack.Name)
	err := b.installFromArchive(buildpack, destination)
	if err == nil {
		return nil
	}

	if _, ok := err.(NotZipFileError); !ok {
		return err
	}

	buildpackURL, err := url.Parse(buildpack.URL)
	if err != nil {
		return fmt.Errorf("invalid buildpack url (%s): %s", buildpack.URL, err.Error())
	}

	return GitClone(*buildpackURL, destination)
}

func (b *BuildpackManager) installFromArchive(buildpack builder.Buildpack, buildpackPath string) error {
	tmpDir, err := ioutil.TempDir("", "buildpacks")
	if err != nil {
		return err
	}

	bytes, err := OpenBuildpackURL(buildpack.URL, b.internalClient)
	if err != nil {
		var err2 error
		bytes, err2 = OpenBuildpackURL(buildpack.URL, b.defaultClient)
		if err2 != nil {
			return errors.Wrap(err, fmt.Sprintf("default client also failed: %s", err2.Error()))
		}
	}

	fileName := filepath.Join(tmpDir, fmt.Sprintf("buildback-%d-.zip", time.Now().Nanosecond()))
	defer func() {
		err = os.Remove(fileName)
	}()

	err = ioutil.WriteFile(fileName, bytes, 0644) //nolint:gosec
	if err != nil {
		return err
	}

	err = os.MkdirAll(buildpackPath, 0777)
	if err != nil {
		return err
	}

	err = b.unzipper.Extract(fileName, buildpackPath)
	if err != nil {
		return NotZipFileError{err: err}
	}

	return err
}

func (b *BuildpackManager) writeBuildpackJSON(buildpacks []builder.Buildpack) error {
	bytes, err := json.Marshal(buildpacks)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(filepath.Join(b.buildpackDir, configFileName), bytes, 0644) //nolint:gosec
	if err != nil {
		return err
	}

	return nil
}
