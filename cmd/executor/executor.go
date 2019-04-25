package main

import (
	"fmt"
	"os"

	"code.cloudfoundry.org/eirini-staging"
	"github.com/pkg/errors"
)

func main() {
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

	commander := &eirinistaging.IOCommander{
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Stdin:  os.Stdin,
	}

	packsConf := eirinistaging.PacksBuilderConf{
		PacksBuilderPath:          packsBuilderPath,
		BuildpacksDir:             buildpacksDir,
		OutputDropletLocation:     outputDropletLocation,
		OutputBuildArtifactsCache: outputBuildArtifactsCache,
		OutputMetadataLocation:    outputMetadataLocation,
	}

	executor := &eirinistaging.PacksExecutor{
		Conf:           packsConf,
		Commander:      commander,
		Extractor:      &eirinistaging.Unzipper{},
		DownloadDir:    downloadDir,
		BuildpacksJSON: buildpackCfg,
	}

	err := executor.ExecuteRecipe()
	if err != nil {
		responder.RespondWithFailure(errors.Wrap(err, "failed to create droplet"))
		os.Exit(1)
	}

	fmt.Println("Recipe Execution completed")
}
