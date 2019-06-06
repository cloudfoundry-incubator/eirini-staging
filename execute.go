package eirinistaging

import (
	"os"

	"code.cloudfoundry.org/eirini-staging/builder"
)

type IOCommander struct {
	Stdout   *os.File
	Stderr   *os.File
	Stdin    *os.File
	ExitCode int
}

// func (c *IOCommander) Exec(cmd string, args ...string) (int, error) {
// 	command := exec.Command(cmd, args...) //#nosec
// 	command.Stdout = c.Stdout
// 	command.Stderr = c.Stderr
// 	command.Stdin = c.Stdin
//
// 	err := command.Run()
// 	return command.ProcessState.ExitCode(), err
// }

type PacksExecutor struct {
	Conf *builder.Config
	// Commander Commander
	// Extractor Extractor
	// DownloadDir    string
	// BuildpacksJSON string
}

func (e *PacksExecutor) ExecuteRecipe() error {
	// e.Conf.BuildDir = buildDir

	// args := []string{
	// 	"-buildDir", buildDir,
	// 	"-buildpacksDir", e.Conf.BuildpacksDir,
	// 	"-outputDroplet", e.Conf.OutputDropletLocation,
	// 	"-outputBuildArtifactsCache", e.Conf.OutputBuildArtifactsCache,
	// 	"-outputMetadata", e.Conf.OutputMetadataLocation,
	// }

	// if e.BuildpacksJSON != "" {
	// 	var buildpacks []Buildpack
	// 	err = json.Unmarshal([]byte(e.BuildpacksJSON), &buildpacks)
	// 	if err != nil {
	// 		return err
	// 	}
	//
	// 	if len(buildpacks) == 1 && buildpacks[0].SkipDetect != nil && *buildpacks[0].SkipDetect {
	// 		e.Conf.SkipDetect = true
	// 		e.Conf.BuildpackOrder = []string{buildpacks[0].Name}
	// 	} else if len(buildpacks) > 0 {
	// 		for _, b := range buildpacks {
	// 			e.Conf.BuildpackOrder = append(e.Conf.BuildpackOrder, buildpacks[0].Name)
	// 		}
	// 	}
	// }

	// exitCode, err := e.Commander.Exec(e.Conf.PacksBuilderPath, args...)
	// if err != nil {
	// 	return ErrorWithExitCode{ExitCode: exitCode, InnerError: err}
	// }

	runner := builder.NewRunner(e.Conf)
	defer runner.CleanUp()

	_, err := runner.Run()
	if err != nil {
		return err
	}

	return nil
}
