package cmd

import (
	"os"
	"path/filepath"

	eirinistaging "code.cloudfoundry.org/eirini-staging"
)

func CreateResponder(certPath string) (eirinistaging.Responder, error) {
	stagingGUID := os.Getenv(eirinistaging.EnvStagingGUID)
	completionCallback := os.Getenv(eirinistaging.EnvCompletionCallback)
	eiriniAddress := os.Getenv(eirinistaging.EnvEiriniAddress)

	cacert := filepath.Join(certPath, eirinistaging.CACertName)
	cert := filepath.Join(certPath, eirinistaging.EiriniClientCert)
	key := filepath.Join(certPath, eirinistaging.EiriniClientKey)

	return eirinistaging.NewResponder(stagingGUID, completionCallback, eiriniAddress, cacert, cert, key)
}
