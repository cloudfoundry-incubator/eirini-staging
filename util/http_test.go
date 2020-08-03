package util_test

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"path"
	"path/filepath"
	"runtime"

	"code.cloudfoundry.org/eirini-staging/util"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func extractSubject(key, cert string) []byte {
	p, err := tls.LoadX509KeyPair(key, cert)
	Expect(err).NotTo(HaveOccurred())
	c, err := x509.ParseCertificate(p.Certificate[0])
	Expect(err).NotTo(HaveOccurred())

	return c.RawSubject
}

var (
	_, b, _, _ = runtime.Caller(0)
	certPath   = path.Join(filepath.Dir(b), "..", "testdata", "certs")
)

var _ = Describe("HTTP", func() {
	Context("CreateTLSHTTPClient", func() {
		var (
			systemCerts  [][]byte
			testSubject1 []byte
			testSubject2 []byte
		)

		BeforeEach(func() {
			systemCertPool, err := x509.SystemCertPool()
			Expect(err).NotTo(HaveOccurred())
			systemCerts = systemCertPool.Subjects()
			testSubject1 = extractSubject(path.Join(certPath, "1.cert"), path.Join(certPath, "1.key"))
			testSubject2 = extractSubject(path.Join(certPath, "2.cert"), path.Join(certPath, "2.key"))
		})

		It("Should include system certificates", func() {
			client, err := util.CreateTLSHTTPClient([]util.CertPaths{})
			Expect(err).NotTo(HaveOccurred())

			config := client.Transport.(*http.Transport).TLSClientConfig
			Expect(config.RootCAs).NotTo(BeNil())
			for _, systemCert := range systemCerts {
				Expect(config.RootCAs.Subjects()).To(ContainElement(systemCert))
			}
		})

		It("Should append certificates", func() {
			client, err := util.CreateTLSHTTPClient([]util.CertPaths{
				{Crt: path.Join(certPath, "1.cert"), Key: path.Join(certPath, "1.key"), Ca: path.Join(certPath, "1.cert")},
				{Crt: path.Join(certPath, "2.cert"), Key: path.Join(certPath, "2.key"), Ca: path.Join(certPath, "2.cert")},
			})
			Expect(err).NotTo(HaveOccurred())

			config := client.Transport.(*http.Transport).TLSClientConfig
			Expect(config.RootCAs).NotTo(BeNil())
			Expect(config.RootCAs.Subjects()).To(ContainElement(testSubject2))
			Expect(config.RootCAs.Subjects()).To(ContainElement(testSubject1))
			Expect(config.RootCAs.Subjects()).To(HaveLen(len(systemCerts) + 2))
		})
	})
})
