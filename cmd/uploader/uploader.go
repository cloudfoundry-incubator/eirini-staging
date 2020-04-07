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
	buildpacksConfig := os.Getenv(eirinistaging.EnvBuildpacks)
	dropletUploadURL := os.Getenv(eirinistaging.EnvDropletUploadURL)
	certPath := util.GetEnvOrDefault(eirinistaging.EnvCertsPath, eirinistaging.CCCertsMountPath)
	dropletLocation := util.GetEnvOrDefault(eirinistaging.EnvOutputDropletLocation, eirinistaging.RecipeOutputDropletLocation)
	metadataLocation := util.GetEnvOrDefault(eirinistaging.EnvOutputMetadataLocation, eirinistaging.RecipeOutputMetadataLocation)
	buildpackCacheLocation := util.GetEnvOrDefault(eirinistaging.EnvOutputBuildArtifactsCache, eirinistaging.RecipeOutputBuildArtifactsCache)
	buildpackCacheUploadURL := util.MustGetEnv(eirinistaging.EnvBuildpackCacheUploadURI)

	responder, err := cmd.CreateResponder(certPath)
	if err != nil {
		log.Fatal("failed to initialize responder", err)
	}

	client, err := createUploaderHTTPClient(certPath)
	if err != nil {
		responder.RespondWithFailure(err)
		os.Exit(1) // nolint:gomnd
	}

	uploadClient := eirinistaging.DropletUploader{
		Client: client,
	}

	err = uploadClient.Upload(dropletUploadURL, dropletLocation)
	if err != nil {
		responder.RespondWithFailure(err)
		log.Fatalf("failed to upload droplet: %s", err.Error())
	}

	if buildpackCacheUploadURL != "" {
		err = uploadClient.Upload(buildpackCacheUploadURL, buildpackCacheLocation)
		if err != nil {
			responder.RespondWithFailure(err)
			log.Fatalf("failed to upload buildpack cache. %s", err.Error())
		}
	}

	resp, err := responder.PrepareSuccessResponse(metadataLocation, buildpacksConfig)
	if err != nil {
		responder.RespondWithFailure(err)
		log.Fatalf("failed to prepare response: %s", err.Error())
	}

	err = responder.RespondWithSuccess(resp)
	if err != nil {
		log.Fatalf("failed to prepare response: %s", err.Error())
	}
}

func createUploaderHTTPClient(certPath string) (*http.Client, error) {
	cacert := filepath.Join(certPath, eirinistaging.CACertName)
	cert := filepath.Join(certPath, eirinistaging.CCAPICertName)
	key := filepath.Join(certPath, eirinistaging.CCAPIKeyName)

	return util.CreateTLSHTTPClient([]util.CertPaths{
		{Crt: cert, Key: key, Ca: cacert},
	})
}
