package main

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	eirinistaging "code.cloudfoundry.org/eirini-staging"
	"code.cloudfoundry.org/eirini-staging/builder"
	"code.cloudfoundry.org/eirini-staging/cmd"
	"code.cloudfoundry.org/eirini-staging/util"
	"github.com/pkg/errors"
)

const (
	ExitReason = "failed to create droplet"
)

func main() {
	buildpackCfg := os.Getenv(eirinistaging.EnvBuildpacks)
	buildpacksDir := util.GetEnvOrDefault(eirinistaging.EnvBuildpacksDir, eirinistaging.RecipeBuildPacksDir)
	outputDropletLocation := util.GetEnvOrDefault(eirinistaging.EnvOutputDropletLocation, eirinistaging.RecipeOutputDropletLocation)
	outputBuildArtifactsCache := util.GetEnvOrDefault(eirinistaging.EnvOutputBuildArtifactsCache, eirinistaging.RecipeOutputBuildArtifactsCache)
	outputMetadataLocation := util.GetEnvOrDefault(eirinistaging.EnvOutputMetadataLocation, eirinistaging.RecipeOutputMetadataLocation)
	cacheDir := util.GetEnvOrDefault(eirinistaging.EnvBuildArtifactsCacheDir, eirinistaging.BuildArtifactsCacheDir)
	downloadDir := util.GetEnvOrDefault(eirinistaging.EnvWorkspaceDir, eirinistaging.RecipeWorkspaceDir)
	certPath := util.GetEnvOrDefault(eirinistaging.EnvCertsPath, eirinistaging.CCCertsMountPath)

	responder, err := cmd.CreateResponder(certPath)
	if err != nil {
		log.Fatal("failed to initialize responder", err)
	}

	buildDir, err := extract(downloadDir)
	if err != nil {
		responder.RespondWithFailure(errors.Wrap(err, ExitReason))
		os.Exit(1) // nolint:gomnd
	}
	defer os.RemoveAll(buildDir)

	buildConfig := builder.Config{
		BuildDir:                  buildDir,
		BuildpacksDir:             buildpacksDir,
		OutputDropletLocation:     outputDropletLocation,
		OutputBuildArtifactsCache: outputBuildArtifactsCache,
		OutputMetadataLocation:    outputMetadataLocation,
		BuildArtifactsCache:       cacheDir,
	}
	if err = buildConfig.InitBuildpacks(buildpackCfg); err != nil {
		responder.RespondWithFailure(errors.Wrap(err, ExitReason))
		os.Exit(1) // nolint:gomnd
	}

	err = execute(&buildConfig)
	if err != nil {
		exitCode := builder.SystemFailCode
		if withExitCode, ok := err.(builder.DescriptiveError); ok {
			exitCode = withExitCode.ExitCode
		}
		responder.RespondWithFailure(errors.Wrap(err, ExitReason))
		os.Exit(exitCode)
	}
}

func execute(conf *builder.Config) error {
	runner := builder.NewRunner(conf)
	defer runner.CleanUp()

	return runner.Run()
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
