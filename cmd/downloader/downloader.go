package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	eirinistaging "code.cloudfoundry.org/eirini-staging"
	"code.cloudfoundry.org/eirini-staging/cmd"
	"code.cloudfoundry.org/eirini-staging/util"
)

func main() {
	log.Println("downloader-started")
	defer log.Println("downloader-done")

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

	responder, err := cmd.CreateResponder(certPath)
	if err != nil {
		log.Fatal("failed to initialize responder", err)
	}
	downloadClient, err := createDownloadHTTPClient(certPath)
	if err != nil {
		responder.RespondWithFailure(err)
		log.Fatalf("error creating http client: %s", err.Error())
	}

	buildpackManager := eirinistaging.NewBuildpackManager(downloadClient, http.DefaultClient, buildpacksDir, buildpacksJSON)
	packageInstaller := eirinistaging.NewPackageManager(downloadClient, appBitsDownloadURL, workspaceDir)

	for _, installer := range []eirinistaging.Installer{
		buildpackManager,
		packageInstaller,
	} {
		if err = installer.Install(); err != nil {
			responder.RespondWithFailure(err)
			log.Fatalf("error installing: %s", err.Error())
		}
	}
}

func createDownloadHTTPClient(certPath string) (*http.Client, error) {
	cacert := filepath.Join(certPath, eirinistaging.CCInternalCACertName)
	cert := filepath.Join(certPath, eirinistaging.CCAPICertName)
	key := filepath.Join(certPath, eirinistaging.CCAPIKeyName)

	return util.CreateTLSHTTPClient([]util.CertPaths{
		{Crt: cert, Key: key, Ca: cacert},
	})
}
