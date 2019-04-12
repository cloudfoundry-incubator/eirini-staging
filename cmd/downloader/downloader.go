package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"code.cloudfoundry.org/eirinistaging"
	"code.cloudfoundry.org/eirinistaging/util"
)

func main() {

	stagingGUID := os.Getenv(eirinistaging.EnvStagingGUID)
	completionCallback := os.Getenv(eirinistaging.EnvCompletionCallback)
	eiriniAddress := os.Getenv(eirinistaging.EnvEiriniAddress)
	appBitsDownloadURL := os.Getenv(eirinistaging.EnvDownloadURL)
	buildpacksJSON := os.Getenv(eirinistaging.EnvBuildpacks)

	buildpacksDir, ok := os.LookupEnv(eirinistaging.EnvBuildpacksDir)
	if !ok {
		buildpacksDir = eirinistaging.RecipeBuildPacksDir
	}

	certPath, ok := os.LookupEnv(eirinistaging.EnvCertsPath)
	if !ok {
		certPath = eirinistaging.CCCertsMountPath
	}

	workspaceDir, ok := os.LookupEnv(eirinistaging.EnvWorkspaceDir)
	if !ok {
		workspaceDir = eirinistaging.RecipeWorkspaceDir
	}

	responder := eirinistaging.NewResponder(stagingGUID, completionCallback, eiriniAddress)

	downloadClient, err := createDownloadHTTPClient(certPath)
	if err != nil {
		fmt.Println(fmt.Sprintf("error creating http client: %s", err))
		responder.RespondWithFailure(err)
		os.Exit(1)
	}

	buildpackManager := eirinistaging.NewBuildpackManager(downloadClient, http.DefaultClient, buildpacksDir, buildpacksJSON)
	packageInstaller := eirinistaging.NewPackageManager(downloadClient, appBitsDownloadURL, workspaceDir)

	for _, installer := range []eirinistaging.Installer{
		buildpackManager,
		packageInstaller,
	} {
		if err = installer.Install(); err != nil {
			responder.RespondWithFailure(err)
			os.Exit(1)
		}
	}

	fmt.Println("Downloading completed")
}

func createDownloadHTTPClient(certPath string) (*http.Client, error) {
	cacert := filepath.Join(certPath, eirinistaging.CCInternalCACertName)
	cert := filepath.Join(certPath, eirinistaging.CCAPICertName)
	key := filepath.Join(certPath, eirinistaging.CCAPIKeyName)

	return util.CreateTLSHTTPClient([]util.CertPaths{
		{Crt: cert, Key: key, Ca: cacert},
	})
}
