package main

import (
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	eirinistaging "code.cloudfoundry.org/eirini-staging"
	"code.cloudfoundry.org/eirini-staging/cmd"
	"code.cloudfoundry.org/eirini-staging/util"
)

func main() {
	log.Println("downloader-started")
	defer log.Println("downloader-done")

	if err := os.MkdirAll("/buildpack/cache", 0755); err != nil {
		panic("cannot create cache dir: " + err.Error())
	}

	if err := os.MkdirAll("/buildpack/tmp", 0755); err != nil {
		panic("cannot create temp dir: " + err.Error())
	}

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

	buildpackCacheDir, ok := os.LookupEnv(eirinistaging.EnvBuildArtifactsCacheDir)
	if !ok {
		buildpackCacheDir = eirinistaging.BuildArtifactsCacheDir
	}

	buildpackCacheURI, ok := os.LookupEnv(eirinistaging.EnvBuildpackCacheDownloadURI)
	if !ok {
		panic("failed to lookup buildpack cache URI")
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

	installers := []eirinistaging.Installer{
		buildpackManager,
		packageInstaller,
	}

	if buildpackCacheURI != "" {
		buildpackCacheInstaller := eirinistaging.NewPackageManager(downloadClient, buildpackCacheURI, buildpackCacheDir)

		installers = append(installers, buildpackCacheInstaller)
	}

	log.Println("Installing dependencies")
	for _, installer := range installers {
		if err = installer.Install(); err != nil {
			responder.RespondWithFailure(err)
			log.Fatalf("error installing: %s", err.Error())
		}
	}

	if buildpackCacheURI != "" {
		cmd := exec.Command("/bin/tar", "-xzf", buildpackCacheDir+"/app.zip")
		cmd.Dir = buildpackCacheDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			log.Fatalf("error unzipping: %s", err.Error())
		}
	}
}

func createDownloadHTTPClient(certPath string) (*http.Client, error) {
	cacert := filepath.Join(certPath, eirinistaging.CACertName)
	cert := filepath.Join(certPath, eirinistaging.CCAPICertName)
	key := filepath.Join(certPath, eirinistaging.CCAPIKeyName)

	return util.CreateTLSHTTPClient([]util.CertPaths{
		{Crt: cert, Key: key, Ca: cacert},
	})
}
