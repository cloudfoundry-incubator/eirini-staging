package main

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	eirinistaging "code.cloudfoundry.org/eirini-staging"
	"code.cloudfoundry.org/eirini-staging/builder"
	"github.com/pkg/errors"
)

func main() {
	log.Println("executor-started")
	defer log.Println("executor-done")

	buildpackCfg := os.Getenv(eirinistaging.EnvBuildpacks)
	stagingGUID := os.Getenv(eirinistaging.EnvStagingGUID)
	completionCallback := os.Getenv(eirinistaging.EnvCompletionCallback)
	eiriniAddress := os.Getenv(eirinistaging.EnvEiriniAddress)
	buildpacksDir, ok := os.LookupEnv(eirinistaging.EnvBuildpacksDir)
	if !ok {
		buildpacksDir = eirinistaging.RecipeBuildPacksDir
	}

	outputDropletLocation, ok := os.LookupEnv(eirinistaging.EnvOutputDropletLocation)
	if !ok {
		outputDropletLocation = eirinistaging.RecipeOutputDropletLocation
	}

	outputBuildArtifactsCache, ok := os.LookupEnv(eirinistaging.EnvOutputBuildArtifactsCache)
	if !ok {
		outputBuildArtifactsCache = eirinistaging.RecipeOutputBuildArtifactsCache
	}

	outputMetadataLocation, ok := os.LookupEnv(eirinistaging.EnvOutputMetadataLocation)
	if !ok {
		outputMetadataLocation = eirinistaging.RecipeOutputMetadataLocation
	}

	packsBuilderPath, ok := os.LookupEnv(eirinistaging.EnvPacksBuilderPath)
	if !ok {
		packsBuilderPath = eirinistaging.RecipePacksBuilderPath
	}

	downloadDir, ok := os.LookupEnv(eirinistaging.EnvWorkspaceDir)
	if !ok {
		downloadDir = eirinistaging.RecipeWorkspaceDir
	}

	responder := eirinistaging.NewResponder(stagingGUID, completionCallback, eiriniAddress)

	// commander := &eirinistaging.IOCommander{
	// 	Stdout: os.Stdout,
	// 	Stderr: os.Stderr,
	// 	Stdin:  os.Stdin,
	// }

	exitReason := "failed to create droplet"
	exitCode := 1

	buildDir, err := extract(downloadDir)
	if err != nil {
		responder.RespondWithFailure(errors.Wrap(err, exitReason))
		os.Exit(exitCode)
	}
	defer cleanup(buildDir)

	buildConfig, err := builder.Config{
		BuildDir:                  buildDir,
		PacksBuilderPath:          packsBuilderPath,
		BuildpacksDir:             buildpacksDir,
		OutputDropletLocation:     outputDropletLocation,
		OutputBuildArtifactsCache: outputBuildArtifactsCache,
		OutputMetadataLocation:    outputMetadataLocation,
	}.Init(downloadDir, buildpackCfg)
	if err != nil {
		responder.RespondWithFailure(errors.Wrap(err, exitReason))
		os.Exit(exitCode)
	}

	executor := &eirinistaging.PacksExecutor{
		Conf: &buildConfig,
		// Commander: commander,
	}

	err = executor.ExecuteRecipe()
	if err != nil {
		exitCode := builder.SystemFailCode
		if withExitCode, ok := err.(builder.DescriptiveError); ok {
			exitCode = withExitCode.ExitCode
		}
		responder.RespondWithFailure(errors.Wrap(err, exitReason))
		os.Exit(exitCode)
	}
}

func extract(downloadDir string) (string, error) {
	extractor := &eirinistaging.Unzipper{}
	buildDir, err := ioutil.TempDir("", "app-bits")
	if err != nil {
		return "", err
	}

	err = extractor.Extract(filepath.Join(downloadDir, eirinistaging.AppBits), buildDir)
	if err != nil {
		return "", err
	}

	return buildDir, err
}

func cleanup(buildDir string) error {
	err := os.RemoveAll(buildDir)
	if err != nil {
		return err
	}

	return nil
}
