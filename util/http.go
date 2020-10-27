package util

import (
	"fmt"
	"net/http"

	"code.cloudfoundry.org/tlsconfig"
)

type CertPaths struct {
	Crt, Key, Ca string
}

func CreateTLSHTTPClient(certPaths []CertPaths) (*http.Client, error) {
	tlsOpts := []tlsconfig.TLSOption{tlsconfig.WithInternalServiceDefaults()}
	poolOpts := []tlsconfig.PoolOption{}

	for _, certPath := range certPaths {
		tlsOpts = append(tlsOpts, tlsconfig.WithIdentityFromFile(certPath.Crt, certPath.Key))
		poolOpts = append(poolOpts, tlsconfig.WithCertsFromFile(certPath.Ca))
	}

	poolBuilder := tlsconfig.FromSystemPool(poolOpts...)
	clientOption := tlsconfig.WithAuthorityBuilder(poolBuilder)
	tlsConfig, err := tlsconfig.Build(tlsOpts...).Client(clientOption)
	if err != nil {
		return nil, fmt.Errorf("failed to build tlsconfig: %w", err)
	}

	return &http.Client{Transport: &http.Transport{TLSClientConfig: tlsConfig}}, nil
}
