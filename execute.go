package eirinistaging

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
)

type IOCommander struct {
	Stdout   *os.File
	Stderr   *os.File
	Stdin    *os.File
	ExitCode int
}

func (c *IOCommander) Exec(cmd string, args ...string) (int, error) {
	command := exec.Command(cmd, args...) //#nosec
	command.Stdout = c.Stdout
	command.Stderr = c.Stderr
	command.Stdin = c.Stdin

	err := command.Run()
	return command.ProcessState.ExitCode(), err
}

type PacksBuilderConf struct {
	PacksBuilderPath          string
	BuildpacksDir             string
	OutputDropletLocation     string
	OutputBuildArtifactsCache string
	OutputMetadataLocation    string
}

type PacksExecutor struct {
	Conf           PacksBuilderConf
	Commander      Commander
	Extractor      Extractor
	DownloadDir    string
	BuildpacksJSON string
}

func (e *PacksExecutor) ExecuteRecipe() error {
	buildDir, err := e.extract()
	if err != nil {
		return err
	}

	args := []string{
		"-buildDir", buildDir,
		"-buildpacksDir", e.Conf.BuildpacksDir,
		"-outputDroplet", e.Conf.OutputDropletLocation,
		"-outputBuildArtifactsCache", e.Conf.OutputBuildArtifactsCache,
		"-outputMetadata", e.Conf.OutputMetadataLocation,
	}

	if e.BuildpacksJSON != "" {
		var buildpacks []Buildpack
		err = json.Unmarshal([]byte(e.BuildpacksJSON), &buildpacks)
		if err != nil {
			return err
		}

		if len(buildpacks) == 1 && buildpacks[0].SkipDetect != nil && *buildpacks[0].SkipDetect {
			args = append(args, []string{
				"-skipDetect",
				"-buildpackOrder", buildpacks[0].Name,
			}...)
		}
	}

	exitCode, err := e.Commander.Exec(e.Conf.PacksBuilderPath, args...)
	if err != nil {
		return ErrorWithExitCode{ExitCode: exitCode, InnerError: err}
	}

	err = os.RemoveAll(buildDir)
	if err != nil {
		return err
	}

	return nil
}

func (e *PacksExecutor) extract() (string, error) {
	buildDir, err := ioutil.TempDir("", "app-bits")
	if err != nil {
		return "", err
	}

	err = e.Extractor.Extract(filepath.Join(e.DownloadDir, AppBits), buildDir)
	if err != nil {
		return "", err
	}

	return buildDir, err
}
