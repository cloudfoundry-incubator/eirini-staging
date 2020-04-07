package main

import (
	"errors"
	"fmt"
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
	tmpDir := util.MustGetEnv("TMPDIR")
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		log.Fatalf("failed to create TMPDIR at %s: %s", tmpDir, err)
	}

	appBitsDownloadURL := os.Getenv(eirinistaging.EnvDownloadURL)
	buildpacksJSON := os.Getenv(eirinistaging.EnvBuildpacks)

	buildpacksDir := util.GetEnvOrDefault(eirinistaging.EnvBuildpacksDir, eirinistaging.RecipeBuildPacksDir)
	certPath := util.GetEnvOrDefault(eirinistaging.EnvCertsPath, eirinistaging.CCCertsMountPath)
	workspaceDir := util.GetEnvOrDefault(eirinistaging.EnvWorkspaceDir, eirinistaging.RecipeWorkspaceDir)

	buildpackCacheDir := util.GetEnvOrDefault(eirinistaging.EnvBuildArtifactsCacheDir, eirinistaging.BuildArtifactsCacheDir)
	if err := os.MkdirAll(buildpackCacheDir, 0755); err != nil {
		log.Fatalf("failed to create buildpack cache dir at %s: %s", buildpackCacheDir, err)
	}

	responder, err := cmd.CreateResponder(certPath)
	if err != nil {
		log.Fatal("failed to initialize responder", err)
	}

	buildpackCacheURI, ok := os.LookupEnv(eirinistaging.EnvBuildpackCacheDownloadURI)
	if !ok {
		responder.RespondWithFailure(errors.New("buildpack-cache-download-uri-not-set"))
		log.Fatal("buildpack cache download uri is not set")
	}

	downloadClient, err := createDownloadHTTPClient(certPath)
	if err != nil {
		responder.RespondWithFailure(err)
		log.Fatalf("error creating http client: %s", err.Error())
	}

	buildpackManager := eirinistaging.NewBuildpackManager(downloadClient, http.DefaultClient, buildpacksDir, buildpacksJSON)
	appBitsInstaller := eirinistaging.NewPackageManager(downloadClient, appBitsDownloadURL, workspaceDir)
	buildpackCacheInstaller := eirinistaging.NewPackageManager(downloadClient, buildpackCacheURI, buildpackCacheDir)

	installers := []eirinistaging.Installer{
		buildpackManager,
		appBitsInstaller,
	}
	if buildpackCacheURI != "" {
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
		if err := untarCache(buildpackCacheDir); err != nil {
			log.Fatalf("error untarring cache: %s", err.Error())
		}
	}

}

func untarCache(buildpackCacheDir string) error {
	tarPath, err := exec.LookPath("tar")
	if err != nil {
		return err
	}

	cacheZip := fmt.Sprintf("%s/app.zip", buildpackCacheDir)
	cmd := exec.Command(tarPath, "-xzf", cacheZip)
	cmd.Dir = buildpackCacheDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func createDownloadHTTPClient(certPath string) (*http.Client, error) {
	cacert := filepath.Join(certPath, eirinistaging.CACertName)
	cert := filepath.Join(certPath, eirinistaging.CCAPICertName)
	key := filepath.Join(certPath, eirinistaging.CCAPIKeyName)

	return util.CreateTLSHTTPClient([]util.CertPaths{
		{Crt: cert, Key: key, Ca: cacert},
	})
}
