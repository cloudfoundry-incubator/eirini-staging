package main

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	eirinistaging "code.cloudfoundry.org/eirini-staging"
	"code.cloudfoundry.org/eirini-staging/builder"
	"code.cloudfoundry.org/eirini-staging/cmd"
	"github.com/pkg/errors"
)

const (
	ExitReason = "failed to create droplet"
)

func main() {
	log.Println("executor-started")
	defer log.Println("executor-done")

	buildpackCfg := os.Getenv(eirinistaging.EnvBuildpacks)
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

	cacheDir, ok := os.LookupEnv(eirinistaging.EnvBuildArtifactsCacheDir)
	if !ok {
		cacheDir = eirinistaging.BuildArtifactsCacheDir
	}

	downloadDir, ok := os.LookupEnv(eirinistaging.EnvWorkspaceDir)
	if !ok {
		downloadDir = eirinistaging.RecipeWorkspaceDir
	}

	certPath, ok := os.LookupEnv(eirinistaging.EnvCertsPath)
	if !ok {
		certPath = eirinistaging.CCCertsMountPath
	}

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

	buildConfig, err := builder.NewConfig(
		buildDir, buildpacksDir,
		outputDropletLocation,
		outputBuildArtifactsCache,
		outputMetadataLocation,
		cacheDir, buildpackCfg,
	)
	if err != nil {
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
