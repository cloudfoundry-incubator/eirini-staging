package util

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"

	"code.cloudfoundry.org/tlsconfig"
)

type CertPaths struct {
	Crt, Key, Ca string
}

func withSystemCertPool() tlsconfig.ClientOption {
	return func(c *tls.Config) error {
		caCertPool, err := x509.SystemCertPool()
		if err != nil {
			return fmt.Errorf("unable to load system cert pool: %v", err)
		}
		c.RootCAs = caCertPool
		return nil
	}
}

func withAdditionalAuthorityFromFile(caPath string) tlsconfig.ClientOption {
	return func(c *tls.Config) error {
		caBytes, err := ioutil.ReadFile(caPath)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %s", caPath, err.Error())
		}
		if c.RootCAs == nil {
			return fmt.Errorf("cannot add to empty cert pool")
		}
		if ok := c.RootCAs.AppendCertsFromPEM(caBytes); !ok {
			return fmt.Errorf("unable to load CA certificate at %s", caPath)
		}
		return nil
	}
}

func CreateTLSHTTPClient(certPaths []CertPaths) (*http.Client, error) {
	tlsOpts := []tlsconfig.TLSOption{tlsconfig.WithInternalServiceDefaults()}
	tlsClientOpts := []tlsconfig.ClientOption{withSystemCertPool()}

	for _, certPath := range certPaths {
		tlsOpts = append(tlsOpts, tlsconfig.WithIdentityFromFile(certPath.Crt, certPath.Key))
		tlsClientOpts = append(tlsClientOpts, withAdditionalAuthorityFromFile(certPath.Ca))
	}

	tlsConfig, err := tlsconfig.Build(tlsOpts...).Client(tlsClientOpts...)
	if err != nil {
		return nil, err
	}

	return &http.Client{Transport: &http.Transport{TLSClientConfig: tlsConfig}}, nil
}
