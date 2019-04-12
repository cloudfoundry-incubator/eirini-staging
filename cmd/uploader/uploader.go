package main

import (
	"net/http"
	"os"
	"path/filepath"

	"code.cloudfoundry.org/eirinistaging"
	"code.cloudfoundry.org/eirinistaging/util"
)

func main() {
	buildpackCfg := os.Getenv(eirinistaging.EnvBuildpacks)
	stagingGUID := os.Getenv(eirinistaging.EnvStagingGUID)
	completionCallback := os.Getenv(eirinistaging.EnvCompletionCallback)
	eiriniAddress := os.Getenv(eirinistaging.EnvEiriniAddress)
	dropletUploadURL := os.Getenv(eirinistaging.EnvDropletUploadURL)

	certPath, ok := os.LookupEnv(eirinistaging.EnvCertsPath)
	if !ok {
		certPath = eirinistaging.CCCertsMountPath
	}

	dropletLocation, ok := os.LookupEnv(eirinistaging.EnvOutputDropletLocation)
	if !ok {
		dropletLocation = eirinistaging.RecipeOutputDropletLocation
	}

	metadataLocation, ok := os.LookupEnv(eirinistaging.EnvOutputMetadataLocation)
	if !ok {
		metadataLocation = eirinistaging.RecipeOutputMetadataLocation
	}

	responder := eirinistaging.NewResponder(stagingGUID, completionCallback, eiriniAddress)

	client, err := createUploaderHTTPClient(certPath)
	if err != nil {
		responder.RespondWithFailure(err)
		os.Exit(1)
	}

	uploadClient := eirinistaging.DropletUploader{
		Client: client,
	}

	err = uploadClient.Upload(dropletUploadURL, dropletLocation)
	if err != nil {
		responder.RespondWithFailure(err)
		os.Exit(1)
	}

	resp, err := responder.PrepareSuccessResponse(metadataLocation, buildpackCfg)
	if err != nil {
		// TODO: log error
		responder.RespondWithFailure(err)
		os.Exit(1)
	}

	err = responder.RespondWithSuccess(resp)
	if err != nil {
		// TODO: log that it didnt go through
		os.Exit(1)
	}
}

func createUploaderHTTPClient(certPath string) (*http.Client, error) {
	cacert := filepath.Join(certPath, eirinistaging.CCInternalCACertName)
	cert := filepath.Join(certPath, eirinistaging.CCAPICertName)
	key := filepath.Join(certPath, eirinistaging.CCAPIKeyName)

	return util.CreateTLSHTTPClient([]util.CertPaths{
		{Crt: cert, Key: key, Ca: cacert},
	})
}
