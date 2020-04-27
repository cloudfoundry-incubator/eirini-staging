package main

import (
	"crypto/sha256"
	"fmt"
	"hash"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	eirinistaging "code.cloudfoundry.org/eirini-staging"
	"code.cloudfoundry.org/eirini-staging/checksum"
	"code.cloudfoundry.org/eirini-staging/cmd"
	"code.cloudfoundry.org/eirini-staging/util"
)

func main() {
	appBitsDownloadURL := os.Getenv(eirinistaging.EnvDownloadURL)
	buildpacksJSON := os.Getenv(eirinistaging.EnvBuildpacks)

	buildpacksDir := util.GetEnvOrDefault(eirinistaging.EnvBuildpacksDir, eirinistaging.RecipeBuildPacksDir)
	certPath := util.GetEnvOrDefault(eirinistaging.EnvCertsPath, eirinistaging.CCCertsMountPath)
	workspaceDir := util.GetEnvOrDefault(eirinistaging.EnvWorkspaceDir, eirinistaging.RecipeWorkspaceDir)

	buildpackCacheDir := util.GetEnvOrDefault(eirinistaging.EnvBuildArtifactsCacheDir, eirinistaging.BuildArtifactsCacheDir)
	if err := os.MkdirAll(buildpackCacheDir, 0755); err != nil {
		log.Fatalf("failed to create buildpack cache dir at %s: %s", buildpackCacheDir, err)
	}
	buildpackCacheURI := util.MustGetEnv(eirinistaging.EnvBuildpackCacheDownloadURI)

	responder, err := cmd.CreateResponder(certPath)
	if err != nil {
		log.Fatal("failed to initialize responder", err)
	}

	downloadClient, err := createDownloadHTTPClient(certPath)
	if err != nil {
		responder.RespondWithFailure(err)
		log.Fatalf("error creating http client: %s", err.Error())
	}

	installers := []eirinistaging.Installer{
		eirinistaging.NewBuildpackManager(downloadClient, http.DefaultClient, buildpacksDir, buildpacksJSON),
		eirinistaging.NewPackageManager(downloadClient, appBitsDownloadURL, workspaceDir, nil),
	}

	if buildpackCacheURI != "" {
		tmpDir := util.MustGetEnv(eirinistaging.EnvBuildpackCacheDir)
		if err := os.MkdirAll(tmpDir, 0755); err != nil {
			log.Fatalf("failed to create tmpDir at %s: %s", tmpDir, err)
		}
		if err := os.Setenv("TMPDIR", tmpDir); err != nil {
			log.Fatalf("failed to set TMPDIR as %s: %s", tmpDir, err)
		}

		buildpackCacheChecksum := util.MustGetEnv(eirinistaging.EnvBuildpackCacheChecksum)
		checksumVerificationAlgorithm := checksumAlgorithm(util.MustGetEnv(eirinistaging.EnvBuildpackCacheChecksumAlgorithm))
		buildpackCacheInstaller := eirinistaging.NewPackageManager(downloadClient, buildpackCacheURI, buildpackCacheDir, verifyingReader(checksumVerificationAlgorithm, buildpackCacheChecksum))
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

func verifyingReader(hash hash.Hash, chksum string) func(io.Reader) io.Reader {
	return func(reader io.Reader) io.Reader {
		return checksum.NewVerifyingReader(reader, hash, chksum)
	}
}

func checksumAlgorithm(algorithm string) hash.Hash {
	if algorithm == "sha256" {
		return sha256.New()
	}

	log.Fatalf("unsupported checksum verification algorithm: %q", algorithm)
	return nil
}
