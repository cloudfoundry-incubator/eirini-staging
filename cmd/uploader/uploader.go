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
	log.Println("uploader-started")
	defer log.Println("uploader-done")

	buildpacksConfig := os.Getenv(eirinistaging.EnvBuildpacks)
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

	responder, err := cmd.CreateResponder(certPath)
	if err != nil {
		log.Fatal("failed to initialize responder", err)
	}

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
		log.Fatalf("failed to upload droplet: %s", err.Error())
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
	cacert := filepath.Join(certPath, eirinistaging.CCInternalCACertName)
	cert := filepath.Join(certPath, eirinistaging.CCAPICertName)
	key := filepath.Join(certPath, eirinistaging.CCAPIKeyName)

	return util.CreateTLSHTTPClient([]util.CertPaths{
		{Crt: cert, Key: key, Ca: cacert},
	})
}
